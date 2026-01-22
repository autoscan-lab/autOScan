// Package export provides report export functionality.
package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felipetrejos/autoscan/internal/domain"
)

// Markdown exports a report to Markdown format.
func Markdown(report domain.RunReport, outputDir string) (string, error) {
	var b strings.Builder

	// Header
	b.WriteString(fmt.Sprintf("# %s - Grading Report\n\n", report.PolicyName))
	b.WriteString(fmt.Sprintf("**Root:** `%s`\n\n", report.Root))
	b.WriteString(fmt.Sprintf("**Generated:** %s\n\n", report.FinishedAt.Format("2006-01-02 15:04:05")))
	b.WriteString(fmt.Sprintf("**Duration:** %dms\n\n", report.Summary.DurationMs))

	// Summary
	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("| Metric | Value |\n"))
	b.WriteString(fmt.Sprintf("|--------|-------|\n"))
	b.WriteString(fmt.Sprintf("| Total Submissions | %d |\n", report.Summary.TotalSubmissions))
	b.WriteString(fmt.Sprintf("| Compile Pass | %d |\n", report.Summary.CompilePass))
	b.WriteString(fmt.Sprintf("| Compile Fail | %d |\n", report.Summary.CompileFail))
	b.WriteString(fmt.Sprintf("| Compile Timeout | %d |\n", report.Summary.CompileTimeout))
	b.WriteString(fmt.Sprintf("| Clean (no issues) | %d |\n", report.Summary.CleanSubmissions))
	b.WriteString(fmt.Sprintf("| With Banned Calls | %d |\n", report.Summary.SubmissionsWithBanned))
	b.WriteString(fmt.Sprintf("| Total Banned Hits | %d |\n", report.Summary.BannedHitsTotal))
	b.WriteString("\n")

	// Top banned functions
	if len(report.Summary.TopBannedFunctions) > 0 {
		b.WriteString("### Top Banned Functions\n\n")
		b.WriteString("| Function | Count |\n")
		b.WriteString("|----------|-------|\n")
		for fn, count := range report.Summary.TopBannedFunctions {
			b.WriteString(fmt.Sprintf("| `%s` | %d |\n", fn, count))
		}
		b.WriteString("\n")
	}

	// Per-submission details
	b.WriteString("## Submissions\n\n")

	for _, r := range report.Results {
		b.WriteString(fmt.Sprintf("### %s\n\n", r.Submission.ID))

		// Compile status
		if r.Compile.OK {
			b.WriteString("- **Compile:** OK\n")
		} else if r.Compile.TimedOut {
			b.WriteString("- **Compile:** TIMEOUT\n")
		} else {
			b.WriteString(fmt.Sprintf("- **Compile:** FAIL (exit %d)\n", r.Compile.ExitCode))
		}

		// Banned calls
		b.WriteString(fmt.Sprintf("- **Banned calls:** %d\n", r.Scan.TotalHits()))

		if r.Scan.TotalHits() > 0 {
			for fn, hits := range r.Scan.HitsByFunction {
				for _, hit := range hits {
					b.WriteString(fmt.Sprintf("  - `%s`: %s:%d\n", fn, hit.File, hit.Line))
				}
			}
		}

		// Compiler output (if failed)
		if !r.Compile.OK && r.Compile.Stderr != "" {
			b.WriteString("\n**Compiler output (first 30 lines):**\n\n")
			b.WriteString("```\n")
			lines := strings.Split(r.Compile.Stderr, "\n")
			maxLines := 30
			if len(lines) < maxLines {
				maxLines = len(lines)
			}
			for i := 0; i < maxLines; i++ {
				b.WriteString(lines[i] + "\n")
			}
			if len(lines) > 30 {
				b.WriteString(fmt.Sprintf("... (%d more lines)\n", len(lines)-30))
			}
			b.WriteString("```\n")
		}

		b.WriteString("\n")
	}

	// Write to file
	filename := filepath.Join(outputDir, "report.md")
	if err := os.WriteFile(filename, []byte(b.String()), 0644); err != nil {
		return "", err
	}

	return filename, nil
}
