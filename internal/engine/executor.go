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

type Executor struct {
	policy       *policy.Policy
	timeout      time.Duration
	binaryDir    string
	outputName   string
	testFilesDir string
	shortNames   bool
}

func NewExecutor(p *policy.Policy, binaryDir string) *Executor {
	return NewExecutorWithOptions(p, binaryDir, false)
}

func NewExecutorWithOptions(p *policy.Policy, binaryDir string, shortNames bool) *Executor {
	outputName := strings.TrimSuffix(p.Compile.SourceFile, ".c")
	if outputName == "" {
		outputName = p.Compile.Output
		if outputName == "" {
			outputName = "a.out"
		}
	}

	home, _ := os.UserHomeDir()
	return &Executor{
		policy:       p,
		timeout:      p.GetRunTimeout(),
		binaryDir:    binaryDir,
		outputName:   outputName,
		testFilesDir: filepath.Join(home, ".config", "autoscan", "test_files"),
		shortNames:   shortNames,
	}
}

func (e *Executor) submissionDirName(sub domain.Submission) string {
	dirName := sub.ID
	if e.shortNames {
		if idx := strings.Index(dirName, "_"); idx > 0 {
			dirName = dirName[:idx]
		}
	}
	return dirName
}

func (e *Executor) GetBinaryPath(sub domain.Submission) string {
	return filepath.Join(e.binaryDir, e.submissionDirName(sub), e.outputName)
}

func (e *Executor) GetSubmissionBinaryDir(sub domain.Submission) string {
	return filepath.Join(e.binaryDir, e.submissionDirName(sub))
}

func (e *Executor) Execute(ctx context.Context, sub domain.Submission, args []string, input string) domain.ExecuteResult {
	binaryPath := e.GetBinaryPath(sub)
	binaryDir := filepath.Dir(binaryPath)
	resolvedArgs := e.resolveTestFilePaths(args)

	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, binaryPath, resolvedArgs...)
	cmd.Dir = binaryDir
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)
	timedOut := timeoutCtx.Err() == context.DeadlineExceeded

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return domain.NewExecuteResult(exitCode >= 0 || err == nil, exitCode, stdout.String(), stderr.String(), duration, timedOut, args, input)
}

func (e *Executor) ExecuteTestCase(ctx context.Context, sub domain.Submission, tc policy.TestCase) domain.ExecuteResult {
	return e.Execute(ctx, sub, tc.Args, tc.Input).WithTestCase(tc.Name, tc.ExpectedExit)
}

func (e *Executor) ExecuteAllTestCases(ctx context.Context, sub domain.Submission) []domain.ExecuteResult {
	if len(e.policy.Run.TestCases) == 0 {
		return nil
	}
	results := make([]domain.ExecuteResult, len(e.policy.Run.TestCases))
	for i, tc := range e.policy.Run.TestCases {
		results[i] = e.ExecuteTestCase(ctx, sub, tc)
	}
	return results
}

func (e *Executor) BinaryExists(sub domain.Submission) bool {
	_, err := os.Stat(e.GetBinaryPath(sub))
	return err == nil
}

func (e *Executor) HasMultiProcess() bool {
	return e.policy.Run.MultiProcess != nil && e.policy.Run.MultiProcess.Enabled
}

func (e *Executor) GetMultiProcessConfig() *policy.MultiProcessConfig {
	return e.policy.Run.MultiProcess
}

func (e *Executor) GetTestScenarios() []policy.MultiProcessScenario {
	if e.policy.Run.MultiProcess == nil {
		return nil
	}
	return e.policy.Run.MultiProcess.TestScenarios
}

func (e *Executor) resolveTestFilePaths(args []string) []string {
	if len(e.policy.TestFiles) == 0 {
		return args
	}

	testFileSet := make(map[string]bool, len(e.policy.TestFiles))
	for _, tf := range e.policy.TestFiles {
		testFileSet[tf] = true
	}

	resolved := make([]string, len(args))
	for i, arg := range args {
		if testFileSet[arg] {
			resolved[i] = filepath.Join(e.testFilesDir, arg)
		} else {
			resolved[i] = arg
		}
	}
	return resolved
}

func (e *Executor) ExecuteMultiProcess(ctx context.Context, sub domain.Submission, onUpdate func(*domain.MultiProcessResult)) *domain.MultiProcessResult {
	return e.executeMultiProcessWithOverrides(ctx, sub, nil, onUpdate)
}

func (e *Executor) ExecuteMultiProcessScenario(ctx context.Context, sub domain.Submission, scenario policy.MultiProcessScenario, onUpdate func(*domain.MultiProcessResult)) *domain.MultiProcessResult {
	return e.executeMultiProcessWithOverrides(ctx, sub, &scenario, onUpdate)
}

func (e *Executor) executeMultiProcessWithOverrides(ctx context.Context, sub domain.Submission, scenario *policy.MultiProcessScenario, onUpdate func(*domain.MultiProcessResult)) *domain.MultiProcessResult {
	config := e.policy.Run.MultiProcess
	if config == nil || len(config.Executables) == 0 {
		return nil
	}

	result := domain.NewMultiProcessResult()
	if scenario != nil {
		result.ScenarioName = scenario.Name
	}
	start := time.Now()

	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, proc := range config.Executables {
		wg.Add(1)

		args := proc.Args
		input := proc.Input
		var expectedExit *int

		if scenario != nil {
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

		args = e.resolveTestFilePaths(args)

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

			binaryDir := e.GetSubmissionBinaryDir(sub)
			binaryPath := filepath.Join(binaryDir, strings.TrimSuffix(proc.SourceFile, ".c"))

			cmd := exec.CommandContext(timeoutCtx, binaryPath, args...)
			cmd.Dir = binaryDir
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

			if procResult.ExpectedExit != nil {
				procResult.Passed = !procResult.TimedOut && procResult.ExitCode == *procResult.ExpectedExit
			} else {
				procResult.Passed = !procResult.TimedOut
			}

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
