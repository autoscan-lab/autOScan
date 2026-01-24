package engine

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/felipetrejos/autoscan/internal/domain"
	"github.com/felipetrejos/autoscan/internal/policy"
)

// Executor handles running compiled binaries.
type Executor struct {
	policy       *policy.Policy
	timeout      time.Duration
	binaryDir    string // Directory where binaries are stored
	outputName   string // Name of the binary (e.g., "a.out")
	testFilesDir string // Directory where test files are bundled
}

// NewExecutor creates a new executor.
func NewExecutor(p *policy.Policy, binaryDir string) *Executor {
	outputName := p.Compile.Output
	if outputName == "" {
		outputName = "a.out"
	}

	// Get test files directory
	home, _ := os.UserHomeDir()
	testFilesDir := filepath.Join(home, ".config", "autoscan", "test_files")

	return &Executor{
		policy:       p,
		timeout:      p.GetRunTimeout(),
		binaryDir:    binaryDir,
		outputName:   outputName,
		testFilesDir: testFilesDir,
	}
}

// GetBinaryPath returns the path to a submission's binary.
func (e *Executor) GetBinaryPath(sub domain.Submission) string {
	return filepath.Join(e.binaryDir, sub.ID, e.outputName)
}

// Execute runs a submission's binary with the given arguments and input.
func (e *Executor) Execute(ctx context.Context, sub domain.Submission, args []string, input string) domain.ExecuteResult {
	binaryPath := e.GetBinaryPath(sub)

	// Create command with timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, binaryPath, args...)

	// Set stdin if input provided
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	// Check for timeout
	timedOut := timeoutCtx.Err() == context.DeadlineExceeded

	// Get exit code
	exitCode := 0
	ok := true
	if err != nil {
		ok = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if timedOut {
			exitCode = -1
		} else {
			exitCode = -1
		}
	}

	return domain.NewExecuteResult(
		ok || exitCode >= 0, // OK if process ran (even with non-zero exit)
		exitCode,
		stdout.String(),
		stderr.String(),
		duration,
		timedOut,
		args,
		input,
	)
}

// ExecuteTestCase runs a submission against a predefined test case.
func (e *Executor) ExecuteTestCase(ctx context.Context, sub domain.Submission, tc policy.TestCase) domain.ExecuteResult {
	result := e.Execute(ctx, sub, tc.Args, tc.Input)
	return result.WithTestCase(tc.Name, tc.ExpectedExit)
}

// ExecuteAllTestCases runs all test cases defined in the policy.
func (e *Executor) ExecuteAllTestCases(ctx context.Context, sub domain.Submission) []domain.ExecuteResult {
	testCases := e.policy.Run.TestCases
	if len(testCases) == 0 {
		return nil
	}

	results := make([]domain.ExecuteResult, len(testCases))
	for i, tc := range testCases {
		results[i] = e.ExecuteTestCase(ctx, sub, tc)
	}

	return results
}

// BinaryExists checks if the binary for a submission exists.
func (e *Executor) BinaryExists(sub domain.Submission) bool {
	binaryPath := e.GetBinaryPath(sub)
	_, err := exec.LookPath(binaryPath)
	if err != nil {
		// Try if it exists as a file
		cmd := exec.Command("test", "-f", binaryPath)
		return cmd.Run() == nil
	}
	return true
}

// HasMultiProcess returns true if the policy has multi-process mode configured.
func (e *Executor) HasMultiProcess() bool {
	return e.policy.Run.MultiProcess != nil && e.policy.Run.MultiProcess.Enabled
}

// GetMultiProcessConfig returns the multi-process configuration.
func (e *Executor) GetMultiProcessConfig() *policy.MultiProcessConfig {
	return e.policy.Run.MultiProcess
}

// GetTestScenarios returns the multi-process test scenarios.
func (e *Executor) GetTestScenarios() []policy.MultiProcessScenario {
	if e.policy.Run.MultiProcess == nil {
		return nil
	}
	return e.policy.Run.MultiProcess.TestScenarios
}

// resolveTestFilePaths resolves any test file references in arguments.
// If an arg matches a bundled test file name, it's replaced with the full path.
func (e *Executor) resolveTestFilePaths(args []string) []string {
	if len(e.policy.TestFiles) == 0 {
		return args
	}

	// Build set of known test file names
	testFileSet := make(map[string]bool)
	for _, tf := range e.policy.TestFiles {
		testFileSet[tf] = true
	}

	resolved := make([]string, len(args))
	for i, arg := range args {
		if testFileSet[arg] {
			// Replace with full path
			resolved[i] = filepath.Join(e.testFilesDir, arg)
		} else {
			resolved[i] = arg
		}
	}
	return resolved
}

// ExecuteMultiProcess runs multiple processes in parallel for a submission.
// The onUpdate callback is called whenever a process produces output or finishes.
func (e *Executor) ExecuteMultiProcess(
	ctx context.Context,
	sub domain.Submission,
	onUpdate func(*domain.MultiProcessResult),
) *domain.MultiProcessResult {
	return e.executeMultiProcessWithOverrides(ctx, sub, nil, onUpdate)
}

// ExecuteMultiProcessScenario runs a multi-process test scenario.
func (e *Executor) ExecuteMultiProcessScenario(
	ctx context.Context,
	sub domain.Submission,
	scenario policy.MultiProcessScenario,
	onUpdate func(*domain.MultiProcessResult),
) *domain.MultiProcessResult {
	return e.executeMultiProcessWithOverrides(ctx, sub, &scenario, onUpdate)
}

func (e *Executor) executeMultiProcessWithOverrides(
	ctx context.Context,
	sub domain.Submission,
	scenario *policy.MultiProcessScenario,
	onUpdate func(*domain.MultiProcessResult),
) *domain.MultiProcessResult {
	config := e.policy.Run.MultiProcess
	if config == nil || len(config.Executables) == 0 {
		return nil
	}

	result := domain.NewMultiProcessResult()
	if scenario != nil {
		result.ScenarioName = scenario.Name
	}
	start := time.Now()

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, proc := range config.Executables {
		wg.Add(1)

		// Determine args and input for this process
		args := proc.Args
		input := proc.Input
		var expectedExit *int

		if scenario != nil {
			// Override with scenario-specific config
			if scenarioArgs, ok := scenario.ProcessArgs[proc.Name]; ok {
				args = scenarioArgs
			}
			if scenarioInput, ok := scenario.ProcessInputs[proc.Name]; ok {
				input = scenarioInput
			}
			if exit, ok := scenario.ExpectedExits[proc.Name]; ok {
				exitCopy := exit
				expectedExit = &exitCopy
			}
		}

		// Resolve test file paths in args
		args = e.resolveTestFilePaths(args)

		// Initialize process result
		procResult := &domain.ProcessResult{
			Name:         proc.Name,
			SourceFile:   proc.SourceFile,
			Running:      true,
			StartedAt:    time.Now(),
			ExpectedExit: expectedExit,
		}

		mu.Lock()
		result.AddProcess(proc.Name, procResult)
		mu.Unlock()

		go func(proc policy.ProcessConfig, args []string, input string, procResult *domain.ProcessResult) {
			defer wg.Done()

			// Apply start delay if configured
			if proc.StartDelayMs > 0 {
				select {
				case <-time.After(time.Duration(proc.StartDelayMs) * time.Millisecond):
				case <-timeoutCtx.Done():
					procResult.TimedOut = true
					procResult.Running = false
					return
				}
			}

			procResult.StartedAt = time.Now()

			// Build binary path - for multi-process, each source file has its own binary
			// Binary is named after source file without .c extension
			binaryName := strings.TrimSuffix(proc.SourceFile, ".c")
			binaryPath := filepath.Join(e.binaryDir, sub.ID, binaryName)

			cmd := exec.CommandContext(timeoutCtx, binaryPath, args...)

			if input != "" {
				cmd.Stdin = strings.NewReader(input)
			}

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			procResult.FinishedAt = time.Now()
			procResult.Duration = procResult.FinishedAt.Sub(procResult.StartedAt)
			procResult.Running = false

			if timeoutCtx.Err() == context.DeadlineExceeded {
				procResult.TimedOut = true
			}

			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					procResult.ExitCode = exitErr.ExitCode()
				} else {
					procResult.ExitCode = -1
				}
			}

			procResult.Stdout = stdout.String()
			procResult.Stderr = stderr.String()

			// Check if passed (matches expected exit code)
			if procResult.ExpectedExit != nil {
				procResult.Passed = !procResult.TimedOut && procResult.ExitCode == *procResult.ExpectedExit
			} else {
				procResult.Passed = !procResult.TimedOut
			}

			// Notify update
			if onUpdate != nil {
				mu.Lock()
				onUpdate(result)
				mu.Unlock()
			}
		}(proc, args, input, procResult)
	}

	wg.Wait()

	result.TotalDuration = time.Since(start)
	result.AllCompleted = true
	result.AllPassed = true
	for _, pr := range result.Processes {
		if pr.TimedOut {
			result.AllCompleted = false
		}
		if !pr.Passed {
			result.AllPassed = false
		}
	}

	return result
}
