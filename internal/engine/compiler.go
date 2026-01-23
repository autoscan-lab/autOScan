package engine

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

	// Build output path - use outputDir if set (for KeepBinaries), otherwise tempDir
	outputName := e.policy.Compile.Output
	if outputName == "" {
		outputName = "a.out"
	}

	baseDir := e.tempDir
	if e.outputDir != "" {
		baseDir = e.outputDir
	}
	outputPath := filepath.Join(baseDir, sub.ID, outputName)

	// Create output directory
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
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

	// Build full paths to source files
	sourceFiles := make([]string, len(sub.CFiles))
	for i, f := range sub.CFiles {
		sourceFiles[i] = filepath.Join(sub.Path, f)
	}

	// Build gcc arguments
	args := e.policy.BuildGCCArgs(sourceFiles, outputPath)

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

	// Build full command for display
	fullCmd := append([]string{e.policy.Compile.GCC}, args...)

	return domain.NewCompileResult(
		err == nil,
		fullCmd,
		exitCode,
		stdout.String(),
		stderr.String(),
		duration,
		timedOut,
	)
}
