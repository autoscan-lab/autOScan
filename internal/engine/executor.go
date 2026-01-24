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

// unescapeInput converts escape sequences in input strings to actual characters
func unescapeInput(s string) string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\t", "\t")
	s = strings.ReplaceAll(s, "\\r", "\r")
	return s
}

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
		// Convert escape sequences to actual characters
		input = unescapeInput(input)
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
				case <-ctx.Done():
					procResult.Killed = true
					procResult.Running = false
					return
				}
			}

			procResult.StartedAt = time.Now()

			binaryDir := e.GetSubmissionBinaryDir(sub)
			binaryPath := filepath.Join(binaryDir, strings.TrimSuffix(proc.SourceFile, ".c"))

			cmd := exec.CommandContext(ctx, binaryPath, args...)
			cmd.Dir = binaryDir
			if input != "" {
				cmd.Stdin = strings.NewReader(unescapeInput(input))
			}

			// Setup streaming pipes
			stdoutPipe, _ := cmd.StdoutPipe()
			stderrPipe, _ := cmd.StderrPipe()

			if err := cmd.Start(); err != nil {
				procResult.Running = false
				procResult.ExitCode = -1
				procResult.Stderr = err.Error()
				return
			}

			// Send initial update that process has started
			if onUpdate != nil {
				mu.Lock()
				result.TotalDuration = time.Since(start)
				computeMultiProcessStatus(result)
				onUpdate(result)
				mu.Unlock()
			}

			// Stream stdout
			var stdoutDone, stderrDone sync.WaitGroup
			stdoutDone.Add(1)
			stderrDone.Add(1)

			go func() {
				defer stdoutDone.Done()
				buf := make([]byte, 1024)
				for {
					n, err := stdoutPipe.Read(buf)
					if n > 0 {
						mu.Lock()
						procResult.Stdout += string(buf[:n])
						result.TotalDuration = time.Since(start)
						computeMultiProcessStatus(result)
						mu.Unlock()
						if onUpdate != nil {
							mu.Lock()
							onUpdate(result)
							mu.Unlock()
						}
					}
					if err != nil {
						break
					}
				}
			}()

			go func() {
				defer stderrDone.Done()
				buf := make([]byte, 1024)
				for {
					n, err := stderrPipe.Read(buf)
					if n > 0 {
						mu.Lock()
						procResult.Stderr += string(buf[:n])
						result.TotalDuration = time.Since(start)
						computeMultiProcessStatus(result)
						mu.Unlock()
						if onUpdate != nil {
							mu.Lock()
							onUpdate(result)
							mu.Unlock()
						}
					}
					if err != nil {
						break
					}
				}
			}()

			stdoutDone.Wait()
			stderrDone.Wait()

			err := cmd.Wait()
			procResult.FinishedAt = time.Now()
			procResult.Duration = procResult.FinishedAt.Sub(procResult.StartedAt)
			procResult.Running = false

			if ctx.Err() == context.Canceled {
				procResult.Killed = true
			}

			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					procResult.ExitCode = exitErr.ExitCode()
				} else {
					procResult.ExitCode = -1
				}
			}

			if procResult.ExpectedExit != nil {
				procResult.Passed = !procResult.Killed && procResult.ExitCode == *procResult.ExpectedExit
			} else {
				procResult.Passed = !procResult.Killed
			}

			if onUpdate != nil {
				mu.Lock()
				result.TotalDuration = time.Since(start)
				computeMultiProcessStatus(result)
				onUpdate(result)
				mu.Unlock()
			}
		}(proc, args, input, procResult)
	}

	wg.Wait()

	result.TotalDuration = time.Since(start)
	computeMultiProcessStatus(result)

	return result
}

// computeMultiProcessStatus updates AllCompleted and AllPassed based on current process states
func computeMultiProcessStatus(result *domain.MultiProcessResult) {
	allDone := true
	allPassed := true
	for _, pr := range result.Processes {
		if pr.Running {
			allDone = false
		}
		if pr.Killed {
			allPassed = false
		} else if !pr.Passed {
			allPassed = false
		}
	}
	result.AllCompleted = allDone
	result.AllPassed = allPassed
}
