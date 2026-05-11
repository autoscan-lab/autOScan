package details

import (
	"fmt"
	"strings"

	"github.com/autoscan-lab/autoscan-engine/pkg/domain"
	"github.com/autoscan-lab/autoscan/internal/tui/components"
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
	case proc.Killed:
		b.WriteString(components.WarningText.Render("[KILLED]"))
	case proc.TimedOut:
		b.WriteString(components.ErrorText.Render("[TIMEOUT]"))
	case proc.Passed:
		b.WriteString(components.SuccessText.Render("[PASS]"))
	default:
		b.WriteString(components.ErrorText.Render("[FAIL]"))
	}
	b.WriteString(components.SubtleText.Render(fmt.Sprintf(" %dms", proc.Duration.Milliseconds())))
	b.WriteString(renderOutputMatchLabel(proc.OutputMatch, len(proc.OutputDiff), ""))
	return b.String()
}

func renderExecuteStatusLine(r domain.ExecuteResult) string {
	var b strings.Builder
	switch {
	case r.TimedOut:
		b.WriteString(components.ErrorText.Render("[TIMEOUT] Execution timed out"))
	case r.Passed:
		b.WriteString(components.SuccessText.Render("[OK]"))
	default:
		b.WriteString(components.WarningText.Render("[FAIL]"))
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
