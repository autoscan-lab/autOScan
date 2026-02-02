package details

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/feli05/autoscan/internal/domain"
	"github.com/feli05/autoscan/internal/tui/components"
	"github.com/feli05/autoscan/internal/tui/styles"
)

func renderOutputMatchLabel(status domain.OutputMatchStatus, diffCount int, diffSuffix string) string {
	switch status {
	case domain.OutputMatchPass:
		return styles.SuccessText.Render(" | Output: PASS")
	case domain.OutputMatchFail:
		return styles.WarningText.Render(fmt.Sprintf(" | Output: CHECK (%d%s)", diffCount, diffSuffix))
	case domain.OutputMatchMissing:
		return styles.ErrorText.Render(" | Output: MISSING")
	default:
		return ""
	}
}

func renderProcessStatusLine(proc *domain.ProcessResult) string {
	var b strings.Builder
	switch {
	case proc.Running:
		b.WriteString(styles.Highlight.Render("[RUNNING]"))
		b.WriteString(styles.SubtleText.Render(" ..."))
	case proc.Killed:
		b.WriteString(styles.WarningText.Render("[KILLED]"))
		b.WriteString(styles.SubtleText.Render(fmt.Sprintf(" %dms", proc.Duration.Milliseconds())))
	case proc.TimedOut:
		b.WriteString(styles.ErrorText.Render("[TIMEOUT]"))
		b.WriteString(styles.SubtleText.Render(fmt.Sprintf(" %dms", proc.Duration.Milliseconds())))
	case proc.ExpectedExit != nil:
		if proc.Passed {
			b.WriteString(styles.SuccessText.Render(fmt.Sprintf("[PASS] exit %d", proc.ExitCode)))
		} else {
			b.WriteString(styles.ErrorText.Render(fmt.Sprintf("[FAIL] exit %d (expected %d)", proc.ExitCode, *proc.ExpectedExit)))
		}
		b.WriteString(styles.SubtleText.Render(fmt.Sprintf(" %dms", proc.Duration.Milliseconds())))
	case proc.ExitCode == 0:
		b.WriteString(styles.SuccessText.Render(fmt.Sprintf("[OK] exit %d", proc.ExitCode)))
		b.WriteString(styles.SubtleText.Render(fmt.Sprintf(" %dms", proc.Duration.Milliseconds())))
	default:
		b.WriteString(styles.WarningText.Render(fmt.Sprintf("[EXIT %d]", proc.ExitCode)))
		b.WriteString(styles.SubtleText.Render(fmt.Sprintf(" %dms", proc.Duration.Milliseconds())))
	}
	b.WriteString(renderOutputMatchLabel(proc.OutputMatch, len(proc.OutputDiff), ""))
	return b.String()
}

func renderExecuteStatusLine(r domain.ExecuteResult) string {
	var b strings.Builder
	if r.TimedOut {
		b.WriteString(styles.ErrorText.Render("[TIMEOUT] Execution timed out"))
	} else if r.ExitCode == 0 {
		b.WriteString(styles.SuccessText.Render(fmt.Sprintf("[OK] exit %d", r.ExitCode)))
	} else {
		b.WriteString(styles.WarningText.Render(fmt.Sprintf("[EXIT %d]", r.ExitCode)))
	}
	b.WriteString(styles.SubtleText.Render(fmt.Sprintf(" %dms", r.Duration.Milliseconds())))
	b.WriteString(renderOutputMatchLabel(r.OutputMatch, len(r.OutputDiff), " diffs"))
	return b.String()
}

func renderDiffLines(diff []domain.DiffLine, contentWidth int) []string {
	if len(diff) == 0 {
		return nil
	}
	lineStyle := lipgloss.NewStyle().Width(contentWidth).MaxWidth(contentWidth)
	lines := make([]string, 0, len(diff))
	for _, d := range diff {
		var prefix string
		switch d.Type {
		case "removed":
			prefix = "- "
		case "added":
			prefix = "+ "
		default:
			prefix = "  "
		}
		plain := prefix + components.SanitizeDisplay(d.Content)
		plain = components.TruncateToWidth(plain, contentWidth)
		switch d.Type {
		case "removed":
			lines = append(lines, styles.ErrorText.Render(lineStyle.Render(plain)))
		case "added":
			lines = append(lines, styles.SuccessText.Render(lineStyle.Render(plain)))
		default:
			lines = append(lines, lineStyle.Render(plain))
		}
	}
	return lines
}

func appendStderrBlock(lines []string, stderr string, contentWidth int) []string {
	lines = append(lines, "")
	lineStyle := lipgloss.NewStyle().Width(contentWidth).MaxWidth(contentWidth)
	lines = append(lines, styles.WarningText.Render(lineStyle.Render("─── stderr ───")))
	lines = append(lines, components.WrapLines(components.SanitizeDisplay(stderr), contentWidth)...)
	return lines
}
