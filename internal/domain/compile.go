package domain

// CompileResult holds the output of compiling a submission.
type CompileResult struct {
	// OK is true if compilation succeeded (exit code 0)
	OK bool

	// Command is the full gcc command that was executed
	Command []string

	// ExitCode is the gcc process exit code
	ExitCode int

	// Stdout is the standard output from gcc
	Stdout string

	// Stderr is the standard error from gcc (usually contains errors/warnings)
	Stderr string

	// Duration is how long compilation took in milliseconds
	DurationMs int64

	// TimedOut is true if compilation was killed due to timeout
	TimedOut bool
}

// NewCompileResult creates a new CompileResult.
func NewCompileResult(ok bool, command []string, exitCode int, stdout, stderr string, durationMs int64, timedOut bool) CompileResult {
	return CompileResult{
		OK:         ok,
		Command:    command,
		ExitCode:   exitCode,
		Stdout:     stdout,
		Stderr:     stderr,
		DurationMs: durationMs,
		TimedOut:   timedOut,
	}
}
