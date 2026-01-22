package domain

import "time"

// SubmissionResult combines all results for a single submission.
type SubmissionResult struct {
	// Submission is the original submission info
	Submission Submission

	// Compile is the compilation result
	Compile CompileResult

	// Scan is the banned call scan result
	Scan ScanResult

	// Status is the overall status for display
	Status SubmissionStatus
}

// SubmissionStatus represents the overall outcome of processing a submission.
type SubmissionStatus string

const (
	StatusPending   SubmissionStatus = "pending"
	StatusRunning   SubmissionStatus = "running"
	StatusClean     SubmissionStatus = "clean"      // Compiled OK, no banned calls
	StatusBanned    SubmissionStatus = "banned"     // Compiled OK, has banned calls
	StatusFailed    SubmissionStatus = "failed"     // Compilation failed
	StatusTimedOut  SubmissionStatus = "timed_out"  // Compilation timed out
	StatusError     SubmissionStatus = "error"      // Other error
)

// NewSubmissionResult creates a new SubmissionResult and computes the status.
func NewSubmissionResult(sub Submission, compile CompileResult, scan ScanResult) SubmissionResult {
	status := StatusClean

	if compile.TimedOut {
		status = StatusTimedOut
	} else if !compile.OK {
		status = StatusFailed
	} else if scan.TotalHits() > 0 {
		status = StatusBanned
	}

	return SubmissionResult{
		Submission: sub,
		Compile:    compile,
		Scan:       scan,
		Status:     status,
	}
}

// RunReport holds the complete results of a grading run.
type RunReport struct {
	// PolicyName is the name of the policy used
	PolicyName string

	// Root is the root folder that was scanned
	Root string

	// StartedAt is when the run started
	StartedAt time.Time

	// FinishedAt is when the run completed
	FinishedAt time.Time

	// Results is the list of all submission results
	Results []SubmissionResult

	// Summary contains aggregate statistics
	Summary SummaryStats
}

// SummaryStats contains aggregate statistics for a run.
type SummaryStats struct {
	// TotalSubmissions is the total number of submissions found
	TotalSubmissions int

	// CompilePass is the number of submissions that compiled successfully
	CompilePass int

	// CompileFail is the number of submissions that failed to compile
	CompileFail int

	// CompileTimeout is the number of submissions that timed out
	CompileTimeout int

	// BannedHitsTotal is the total number of banned call hits
	BannedHitsTotal int

	// SubmissionsWithBanned is the number of submissions with at least one banned call
	SubmissionsWithBanned int

	// CleanSubmissions is the number of submissions with no issues
	CleanSubmissions int

	// TopBannedFunctions maps function names to their total occurrence count
	TopBannedFunctions map[string]int

	// DurationMs is the total run duration in milliseconds
	DurationMs int64
}

// NewRunReport creates a RunReport and computes summary statistics.
func NewRunReport(policyName, root string, startedAt, finishedAt time.Time, results []SubmissionResult) RunReport {
	summary := computeSummary(results, finishedAt.Sub(startedAt).Milliseconds())

	return RunReport{
		PolicyName: policyName,
		Root:       root,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Results:    results,
		Summary:    summary,
	}
}

func computeSummary(results []SubmissionResult, durationMs int64) SummaryStats {
	stats := SummaryStats{
		TotalSubmissions:   len(results),
		TopBannedFunctions: make(map[string]int),
		DurationMs:         durationMs,
	}

	for _, r := range results {
		switch {
		case r.Compile.TimedOut:
			stats.CompileTimeout++
		case !r.Compile.OK:
			stats.CompileFail++
		default:
			stats.CompilePass++
		}

		if r.Scan.TotalHits() > 0 {
			stats.SubmissionsWithBanned++
			stats.BannedHitsTotal += r.Scan.TotalHits()

			for fn, hits := range r.Scan.HitsByFunction {
				stats.TopBannedFunctions[fn] += len(hits)
			}
		}

		if r.Status == StatusClean {
			stats.CleanSubmissions++
		}
	}

	return stats
}
