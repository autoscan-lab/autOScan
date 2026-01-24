package domain

import "time"

type ExecuteResult struct {
	OK           bool
	ExitCode     int
	Stdout       string
	Stderr       string
	Duration     time.Duration
	TimedOut     bool
	Args         []string
	Input        string
	TestCaseName string
	ExpectedExit *int
	Passed       bool
}

func NewExecuteResult(ok bool, exitCode int, stdout, stderr string, duration time.Duration, timedOut bool, args []string, input string) ExecuteResult {
	return ExecuteResult{
		OK: ok, ExitCode: exitCode, Stdout: stdout, Stderr: stderr,
		Duration: duration, TimedOut: timedOut, Args: args, Input: input,
		Passed: ok && !timedOut,
	}
}

func (r ExecuteResult) WithTestCase(name string, expectedExit *int) ExecuteResult {
	r.TestCaseName = name
	r.ExpectedExit = expectedExit
	if expectedExit != nil {
		r.Passed = r.OK && !r.TimedOut && r.ExitCode == *expectedExit
	}
	return r
}

type MultiProcessResult struct {
	Processes     map[string]*ProcessResult
	Order         []string
	TotalDuration time.Duration
	AllCompleted  bool
	AllPassed     bool
	ScenarioName  string
}

type ProcessResult struct {
	Name         string
	SourceFile   string
	ExitCode     int
	Stdout       string
	Stderr       string
	Duration     time.Duration
	TimedOut     bool
	Killed       bool
	Running      bool
	StartedAt    time.Time
	FinishedAt   time.Time
	ExpectedExit *int
	Passed       bool
}

func NewMultiProcessResult() *MultiProcessResult {
	return &MultiProcessResult{Processes: make(map[string]*ProcessResult), Order: []string{}}
}

func (m *MultiProcessResult) AddProcess(name string, result *ProcessResult) {
	if _, exists := m.Processes[name]; !exists {
		m.Order = append(m.Order, name)
	}
	m.Processes[name] = result
}
