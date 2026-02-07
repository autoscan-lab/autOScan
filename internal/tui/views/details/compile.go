package details

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/felitrejos/autoscan/internal/tui/components"
)

func renderCompileTab(s State) string {
	r := s.Result
	var b strings.Builder

	availableWidth := s.Width - 12
	if availableWidth < 60 {
		availableWidth = 60
	}

	if r.Compile.OK {
		b.WriteString(components.SuccessText.Render("[PASS] Compilation successful"))
	} else if r.Compile.TimedOut {
		b.WriteString(components.ErrorText.Render("[TIMEOUT] Compilation timed out (5s limit)"))
	} else {
		b.WriteString(components.ErrorText.Render(fmt.Sprintf("[FAIL] Compilation failed (exit %d)", r.Compile.ExitCode)))
	}
	b.WriteString("\n\n")

	b.WriteString(components.SubtleText.Render("Command:"))
	b.WriteString("\n")
	if len(r.Compile.Command) > 0 {
		var truncatedCmd []string
		for _, arg := range r.Compile.Command {
			truncatedCmd = append(truncatedCmd, components.TruncatePathToFilename(arg))
		}
		cmd := strings.Join(truncatedCmd, " ")
		cmdStyle := lipgloss.NewStyle().Width(availableWidth)
		b.WriteString(cmdStyle.Render(cmd))
		b.WriteString("\n")
	}

	if r.Compile.Stderr != "" {
		b.WriteString("\n")
		b.WriteString(components.SubtleText.Render("Output:"))
		b.WriteString("\n")
		truncatedStderr := components.TruncatePathsInText(r.Compile.Stderr)
		lines := strings.Split(truncatedStderr, "\n")
		start := s.DetailScroll
		visibleLines := (s.Height - 20)
		if visibleLines < 15 {
			visibleLines = 15
		}
		end := start + visibleLines
		if end > len(lines) {
			end = len(lines)
		}
		if start >= len(lines) {
			start = 0
		}

		lineStyle := lipgloss.NewStyle().Width(availableWidth)
		for i := start; i < end; i++ {
			line := lines[i]
			wrapped := lineStyle.Render(line)
			b.WriteString(wrapped)
			b.WriteString("\n")
		}
		if len(lines) > visibleLines {
			b.WriteString(components.SubtleText.Render(fmt.Sprintf("\n(Showing %d-%d of %d lines, ↑/↓ to scroll)", start+1, end, len(lines))))
		}
	}

	return b.String()
}
