package submissions

import (
	"strings"

	"github.com/feli05/autoscan/internal/domain"
	"github.com/feli05/autoscan/internal/policy"
)

const (
	FilterAll = iota
	FilterFailed
	FilterBanned
	FilterClean
)

func FilterResults(results []domain.SubmissionResult, filter int, query string) []domain.SubmissionResult {
	if results == nil {
		return nil
	}

	var filtered []domain.SubmissionResult
	query = strings.ToLower(strings.TrimSpace(query))

	for _, r := range results {
		switch filter {
		case FilterFailed:
			if !r.Compile.OK {
				if query == "" || strings.Contains(strings.ToLower(r.Submission.ID), query) {
					filtered = append(filtered, r)
				}
			}
		case FilterBanned:
			if r.Scan.TotalHits() > 0 {
				if query == "" || strings.Contains(strings.ToLower(r.Submission.ID), query) {
					filtered = append(filtered, r)
				}
			}
		case FilterClean:
			if r.Status == domain.StatusClean {
				if query == "" || strings.Contains(strings.ToLower(r.Submission.ID), query) {
					filtered = append(filtered, r)
				}
			}
		default:
			if query == "" || strings.Contains(strings.ToLower(r.Submission.ID), query) {
				filtered = append(filtered, r)
			}
		}
	}

	return filtered
}

func InitSimilarityProcesses(pol *policy.Policy) []string {
	if pol == nil {
		return nil
	}

	mp := pol.Run.MultiProcess
	if mp != nil && mp.Enabled && len(mp.Executables) > 0 {
		names := make([]string, 0, len(mp.Executables))
		for _, proc := range mp.Executables {
			names = append(names, proc.Name)
		}
		return names
	}

	// Single-process mode: use source file name or "main"
	name := pol.Compile.SourceFile
	if name == "" {
		name = "main"
	}
	return []string{name}
}

func ResolveSourceFile(pol *policy.Policy, process string) string {
	if pol == nil {
		return ""
	}

	if mp := pol.Run.MultiProcess; mp != nil && mp.Enabled && len(mp.Executables) > 0 {
		for _, proc := range mp.Executables {
			if proc.Name == process {
				return proc.SourceFile
			}
		}
		return ""
	}
	return pol.Compile.SourceFile
}

func CurrentProcessName(processNames []string, selectedProc int) string {
	if len(processNames) == 0 {
		return ""
	}
	if selectedProc < 0 || selectedProc >= len(processNames) {
		return processNames[0]
	}
	return processNames[selectedProc]
}
