package domain

import "time"

// ExecuteResult holds the output of running a compiled submission.
type ExecuteResult struct {
	// OK is true if execution completed (regardless of exit code)
	OK bool

	// ExitCode is the process exit code
	ExitCode int

	// Stdout is the standard output from the program
	Stdout string

	// Stderr is the standard error from the program
	Stderr string

	// Duration is how long execution took
	Duration time.Duration

	// TimedOut is true if execution was killed due to timeout
	TimedOut bool

	// Args are the command-line arguments used
	Args []string

	// Input is the stdin input provided (if any)
	Input string

	// TestCaseName is the name of the test case (if from preset)
	TestCaseName string

	// ExpectedExit is the expected exit code (for test case validation)
	ExpectedExit *int

	// Passed is true if the execution matched expected exit code
	Passed bool
}

// NewExecuteResult creates a new ExecuteResult.
func NewExecuteResult(
	ok bool,
	exitCode int,
	stdout, stderr string,
	duration time.Duration,
	timedOut bool,
	args []string,
	input string,
) ExecuteResult {
	return ExecuteResult{
		OK:       ok,
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
		Duration: duration,
		TimedOut: timedOut,
		Args:     args,
		Input:    input,
		Passed:   ok && !timedOut, // By default, passed if completed without timeout
	}
}

// WithTestCase sets the test case metadata and validates against expected exit.
func (r ExecuteResult) WithTestCase(name string, expectedExit *int) ExecuteResult {
	r.TestCaseName = name
	r.ExpectedExit = expectedExit
	if expectedExit != nil {
		r.Passed = r.OK && !r.TimedOut && r.ExitCode == *expectedExit
	}
	return r
}

// MultiProcessResult holds results from running multiple processes in parallel.
type MultiProcessResult struct {
	// Processes contains results for each process, keyed by process name
	Processes map[string]*ProcessResult

	// Order preserves the original order of process names
	Order []string

	// TotalDuration is how long the entire multi-process run took
	TotalDuration time.Duration

	// AllCompleted is true if all processes finished (vs timeout)
	AllCompleted bool

	// AllPassed is true if all processes passed their expected exit checks
	AllPassed bool

	// ScenarioName is the name of the test scenario (if running a scenario)
	ScenarioName string
}

// ProcessResult holds the result for a single process in a multi-process run.
type ProcessResult struct {
	// Name is the display name (e.g., "Producer")
	Name string

	// SourceFile is the source that was compiled
	SourceFile string

	// ExitCode is the process exit code
	ExitCode int

	// Stdout is the captured output (may be streaming)
	Stdout string

	// Stderr is the captured error output
	Stderr string

	// Duration is how long this process ran
	Duration time.Duration

	// TimedOut is true if this process was killed due to timeout
	TimedOut bool

	// Running is true if the process is still running (for live updates)
	Running bool

	// StartedAt is when the process started
	StartedAt time.Time

	// FinishedAt is when the process finished (zero if still running)
	FinishedAt time.Time

	// ExpectedExit is the expected exit code (nil if not checking)
	ExpectedExit *int

	// Passed is true if process finished and matched expected exit code
	Passed bool
}

// NewMultiProcessResult creates a new multi-process result.
func NewMultiProcessResult() *MultiProcessResult {
	return &MultiProcessResult{
		Processes: make(map[string]*ProcessResult),
		Order:     []string{},
	}
}

// AddProcess adds a process result.
func (m *MultiProcessResult) AddProcess(name string, result *ProcessResult) {
	if _, exists := m.Processes[name]; !exists {
		m.Order = append(m.Order, name)
	}
	m.Processes[name] = result
}
