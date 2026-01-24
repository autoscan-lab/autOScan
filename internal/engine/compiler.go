package engine

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/felipetrejos/autoscan/internal/domain"
	"github.com/felipetrejos/autoscan/internal/policy"
)

const (
	// DefaultTimeout is the default compilation timeout
	DefaultTimeout = 5 * time.Second

	// DefaultWorkers is the default number of parallel workers
	DefaultWorkers = 0 // 0 means use runtime.NumCPU()
)

// CompileEngine handles parallel compilation of submissions.
type CompileEngine struct {
	policy    *policy.Policy
	timeout   time.Duration
	workers   int
	tempDir   string
	outputDir string // If set, binaries are saved here instead of tempDir
}

// CompileOption configures the compile engine.
type CompileOption func(*CompileEngine)

// WithTimeout sets the compilation timeout.
func WithTimeout(d time.Duration) CompileOption {
	return func(e *CompileEngine) {
		e.timeout = d
	}
}

// WithWorkers sets the number of parallel workers.
func WithWorkers(n int) CompileOption {
	return func(e *CompileEngine) {
		e.workers = n
	}
}

// WithOutputDir sets a persistent output directory for binaries.
// When set, binaries are saved to this directory and not cleaned up.
func WithOutputDir(dir string) CompileOption {
	return func(e *CompileEngine) {
		e.outputDir = dir
	}
}

// NewCompileEngine creates a new compile engine.
func NewCompileEngine(p *policy.Policy, opts ...CompileOption) (*CompileEngine, error) {
	// Create temp directory for output binaries
	tempDir, err := os.MkdirTemp("", "autoscan-*")
	if err != nil {
		return nil, err
	}

	e := &CompileEngine{
		policy:  p,
		timeout: DefaultTimeout,
		workers: DefaultWorkers,
		tempDir: tempDir,
	}

	for _, opt := range opts {
		opt(e)
	}

	if e.workers <= 0 {
		e.workers = runtime.NumCPU()
	}

	return e, nil
}

// Cleanup removes the temp directory.
func (e *CompileEngine) Cleanup() error {
	if e.tempDir != "" {
		return os.RemoveAll(e.tempDir)
	}
	return nil
}

// CompileResult wraps a submission with its compile result.
type compileJob struct {
	submission domain.Submission
	result     domain.CompileResult
}

// CompileAll compiles all submissions in parallel.
// The callback is called for each completed compilation (for progress updates).
func (e *CompileEngine) CompileAll(ctx context.Context, submissions []domain.Submission, onComplete func(domain.Submission, domain.CompileResult)) []domain.CompileResult {
	results := make([]domain.CompileResult, len(submissions))

	// Create job channel
	jobs := make(chan int, len(submissions))
	for i := range submissions {
		jobs <- i
	}
	close(jobs)

	// Create worker pool
	var wg sync.WaitGroup
	var mu sync.Mutex

	for w := 0; w < e.workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				sub := submissions[idx]
				result := e.compile(ctx, sub)

				mu.Lock()
				results[idx] = result
				mu.Unlock()

				if onComplete != nil {
					onComplete(sub, result)
				}
			}
		}()
	}

	wg.Wait()
	return results
}

// Compile compiles a single submission.
func (e *CompileEngine) Compile(ctx context.Context, sub domain.Submission) domain.CompileResult {
	return e.compile(ctx, sub)
}

func (e *CompileEngine) compile(ctx context.Context, sub domain.Submission) domain.CompileResult {
	start := time.Now()

	baseDir := e.tempDir
	if e.outputDir != "" {
		baseDir = e.outputDir
	}

	// Create output directory
	outputDir := filepath.Join(baseDir, sub.ID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return domain.NewCompileResult(
			false,
			nil,
			-1,
			"",
			err.Error(),
			time.Since(start).Milliseconds(),
			false,
		)
	}

	// Check if multi-process mode is enabled
	if e.policy.Run.MultiProcess != nil && e.policy.Run.MultiProcess.Enabled && len(e.policy.Run.MultiProcess.Executables) > 0 {
		return e.compileMultiProcess(ctx, sub, outputDir, start)
	}

	// Standard single-binary compilation
	outputName := e.policy.Compile.Output
	if outputName == "" {
		outputName = "a.out"
	}
	outputPath := filepath.Join(outputDir, outputName)

	// Build full paths to source files
	sourceFiles := make([]string, len(sub.CFiles))
	for i, f := range sub.CFiles {
		sourceFiles[i] = filepath.Join(sub.Path, f)
	}

	// Resolve library files from the bundled libraries directory
	var libraryFiles []string
	var libDir string
	var libraryWarnings []string
	if len(e.policy.LibraryFiles) > 0 {
		// Get libraries directory
		home, _ := os.UserHomeDir()
		libDir = filepath.Join(home, ".config", "autoscan", "libraries")

		for _, libFile := range e.policy.LibraryFiles {
			// If it's just a filename, resolve from libraries dir
			// If it's an absolute path, use as-is (backward compatibility)
			var fullPath string
			if filepath.IsAbs(libFile) {
				fullPath = libFile
			} else {
				fullPath = filepath.Join(libDir, libFile)
			}

			// Check if file exists and is readable
			info, err := os.Stat(fullPath)
			if err != nil {
				// File not found - collect warning
				libraryWarnings = append(libraryWarnings, fmt.Sprintf("Warning: Library file not found: %s", fullPath))
				continue
			}

			// For .o files, verify it's actually an object file (has reasonable size)
			if strings.HasSuffix(fullPath, ".o") {
				if info.Size() == 0 {
					libraryWarnings = append(libraryWarnings, fmt.Sprintf("Warning: Object file is empty: %s", fullPath))
					continue
				}
				// Object files should be at least a few bytes (ELF header is ~52 bytes)
				if info.Size() < 50 {
					libraryWarnings = append(libraryWarnings, fmt.Sprintf("Warning: Object file seems corrupted (too small, %d bytes): %s", info.Size(), fullPath))
					// Still include it, but warn
				}
			}

			libraryFiles = append(libraryFiles, fullPath)
		}
	}

	// Build gcc arguments
	// Note: GCC automatically handles compilation and linking in one step.
	// When you pass both .c files and .o files, GCC will:
	// 1. Compile .c files to .o (implicitly)
	// 2. Link all .o files together
	// This is equivalent to the two-step process (gcc -c then gcc -o) but more efficient.
	args := e.policy.BuildGCCArgs(sourceFiles, libraryFiles, outputPath)

	// Add include path for the libraries directory so #include can find .h files
	// The -I flag must come before source files for proper header resolution
	if libDir != "" {
		args = append([]string{"-I", libDir}, args...)
	}

	// Create command with timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, e.policy.Compile.GCC, args...)
	cmd.Dir = sub.Path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run compilation
	err := cmd.Run()
	duration := time.Since(start).Milliseconds()

	// Check for timeout
	timedOut := timeoutCtx.Err() == context.DeadlineExceeded

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if timedOut {
			exitCode = -1
		} else {
			exitCode = -1
		}
	}

	// Prepend library warnings to stderr if any
	stderrStr := stderr.String()
	if len(libraryWarnings) > 0 {
		warnings := strings.Join(libraryWarnings, "\n") + "\n"
		if stderrStr != "" {
			stderrStr = warnings + "─── Compilation Output ───\n" + stderrStr
		} else {
			stderrStr = warnings
		}
	}

	// Build full command for display
	fullCmd := append([]string{e.policy.Compile.GCC}, args...)

	return domain.NewCompileResult(
		err == nil,
		fullCmd,
		exitCode,
		stdout.String(),
		stderrStr,
		duration,
		timedOut,
	)
}

// compileMultiProcess compiles each source file separately for multi-process mode.
// Each process config specifies a source_file, and we compile that to a binary named without .c.
func (e *CompileEngine) compileMultiProcess(ctx context.Context, sub domain.Submission, outputDir string, start time.Time) domain.CompileResult {
	mp := e.policy.Run.MultiProcess

	// Resolve library files
	var libraryFiles []string
	var libDir string
	if len(e.policy.LibraryFiles) > 0 {
		home, _ := os.UserHomeDir()
		libDir = filepath.Join(home, ".config", "autoscan", "libraries")

		for _, libFile := range e.policy.LibraryFiles {
			var fullPath string
			if filepath.IsAbs(libFile) {
				fullPath = libFile
			} else {
				fullPath = filepath.Join(libDir, libFile)
			}
			if _, err := os.Stat(fullPath); err == nil {
				libraryFiles = append(libraryFiles, fullPath)
			}
		}
	}

	var allStdout, allStderr bytes.Buffer
	var allCmds []string
	allOK := true
	exitCode := 0

	// Compile each executable defined in multi-process config
	for _, proc := range mp.Executables {
		// Find the source file in the submission
		sourceFile := filepath.Join(sub.Path, proc.SourceFile)

		// Check if source file exists
		if _, err := os.Stat(sourceFile); err != nil {
			allStderr.WriteString(fmt.Sprintf("Source file not found: %s\n", proc.SourceFile))
			allOK = false
			exitCode = 1
			continue
		}

		// Binary name is source file without .c extension
		binaryName := proc.SourceFile
		if ext := filepath.Ext(binaryName); ext == ".c" {
			binaryName = binaryName[:len(binaryName)-len(ext)]
		}
		outputPath := filepath.Join(outputDir, binaryName)

		// Build args for this single file
		args := []string{}
		args = append(args, e.policy.Compile.Flags...)

		// Add include path for libraries
		if libDir != "" {
			args = append(args, "-I", libDir)
		}

		// Add source file
		args = append(args, sourceFile)

		// Add library files (only .c and .o, not .h)
		for _, libFile := range libraryFiles {
			if filepath.Ext(libFile) == ".c" || filepath.Ext(libFile) == ".o" {
				args = append(args, libFile)
			}
		}

		// Add output
		args = append(args, "-o", outputPath)

		// Create command
		timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
		cmd := exec.CommandContext(timeoutCtx, e.policy.Compile.GCC, args...)
		cmd.Dir = sub.Path

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		cancel()

		// Record command
		fullCmd := append([]string{e.policy.Compile.GCC}, args...)
		allCmds = append(allCmds, fullCmd...)
		allCmds = append(allCmds, ";")

		// Collect output
		if stdout.Len() > 0 {
			allStdout.WriteString(fmt.Sprintf("=== %s ===\n", proc.Name))
			allStdout.Write(stdout.Bytes())
		}
		if stderr.Len() > 0 {
			allStderr.WriteString(fmt.Sprintf("=== %s ===\n", proc.Name))
			allStderr.Write(stderr.Bytes())
		}

		if err != nil {
			allOK = false
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}
	}

	duration := time.Since(start).Milliseconds()

	return domain.NewCompileResult(
		allOK,
		allCmds,
		exitCode,
		allStdout.String(),
		allStderr.String(),
		duration,
		false,
	)
}
