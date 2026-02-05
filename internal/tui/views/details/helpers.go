package details

import (
	"fmt"
	"strings"

	"github.com/feli05/autoscan/internal/domain"
	"github.com/feli05/autoscan/internal/tui/components"
)

func renderOutputMatchLabel(status domain.OutputMatchStatus, diffCount int, diffSuffix string) string {
	switch status {
	case domain.OutputMatchPass:
		return components.SuccessText.Render(" | Output: PASS")
	case domain.OutputMatchFail:
		return components.WarningText.Render(fmt.Sprintf(" | Output: CHECK (%d%s)", diffCount, diffSuffix))
	case domain.OutputMatchMissing:
		return components.ErrorText.Render(" | Output: MISSING")
	default:
		return ""
	}
}

func renderProcessStatusLine(proc *domain.ProcessResult) string {
	var b strings.Builder
	switch {
	case proc.Running:
		b.WriteString(components.Highlight.Render("[RUNNING]"))
		b.WriteString(components.SubtleText.Render(" ..."))
	case proc.Killed:
		b.WriteString(components.WarningText.Render("[KILLED]"))
		b.WriteString(components.SubtleText.Render(fmt.Sprintf(" %dms", proc.Duration.Milliseconds())))
	case proc.TimedOut:
		b.WriteString(components.ErrorText.Render("[TIMEOUT]"))
		b.WriteString(components.SubtleText.Render(fmt.Sprintf(" %dms", proc.Duration.Milliseconds())))
	case proc.ExpectedExit != nil:
		if proc.Passed {
			b.WriteString(components.SuccessText.Render(fmt.Sprintf("[PASS] exit %d", proc.ExitCode)))
		} else {
			b.WriteString(components.ErrorText.Render(fmt.Sprintf("[FAIL] exit %d (expected %d)", proc.ExitCode, *proc.ExpectedExit)))
		}
		b.WriteString(components.SubtleText.Render(fmt.Sprintf(" %dms", proc.Duration.Milliseconds())))
	case proc.ExitCode == 0:
		b.WriteString(components.SuccessText.Render(fmt.Sprintf("[OK] exit %d", proc.ExitCode)))
		b.WriteString(components.SubtleText.Render(fmt.Sprintf(" %dms", proc.Duration.Milliseconds())))
	default:
		b.WriteString(components.WarningText.Render(fmt.Sprintf("[EXIT %d]", proc.ExitCode)))
		b.WriteString(components.SubtleText.Render(fmt.Sprintf(" %dms", proc.Duration.Milliseconds())))
	}
	b.WriteString(renderOutputMatchLabel(proc.OutputMatch, len(proc.OutputDiff), ""))
	return b.String()
}

func renderExecuteStatusLine(r domain.ExecuteResult) string {
	var b strings.Builder
	if r.TimedOut {
		b.WriteString(components.ErrorText.Render("[TIMEOUT] Execution timed out"))
	} else if r.ExitCode == 0 {
		b.WriteString(components.SuccessText.Render(fmt.Sprintf("[OK] exit %d", r.ExitCode)))
	} else {
		b.WriteString(components.WarningText.Render(fmt.Sprintf("[EXIT %d]", r.ExitCode)))
	}
	b.WriteString(components.SubtleText.Render(fmt.Sprintf(" %dms", r.Duration.Milliseconds())))
	b.WriteString(renderOutputMatchLabel(r.OutputMatch, len(r.OutputDiff), " diffs"))
	return b.String()
}

func renderDiffLines(diff []domain.DiffLine, contentWidth int) []string {
	if len(diff) == 0 {
		return nil
	}
	lineStyle := components.FixedWidthStyle(contentWidth)
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
			lines = append(lines, components.ErrorText.Render(lineStyle.Render(plain)))
		case "added":
			lines = append(lines, components.SuccessText.Render(lineStyle.Render(plain)))
		default:
			lines = append(lines, lineStyle.Render(plain))
		}
	}
	return lines
}

func appendStderrBlock(lines []string, stderr string, contentWidth int) []string {
	lines = append(lines, "")
	lineStyle := components.FixedWidthStyle(contentWidth)
	lines = append(lines, components.WarningText.Render(lineStyle.Render("─── stderr ───")))
	lines = append(lines, components.WrapLines(components.SanitizeDisplay(stderr), contentWidth)...)
	return lines
}
