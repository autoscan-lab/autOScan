package components

import (
	"fmt"
	"strings"

	"github.com/felipetrejos/autoscan/internal/tui/styles"
)

func WrapLines(text string, width int) []string {
	if text == "" {
		return nil
	}
	var wrapped []string
	for _, line := range strings.Split(text, "\n") {
		if len(line) <= width {
			wrapped = append(wrapped, line)
		} else {
			for len(line) > width {
				wrapped = append(wrapped, line[:width])
				line = line[width:]
			}
			if len(line) > 0 {
				wrapped = append(wrapped, line)
			}
		}
	}
	return wrapped
}

func ScrollIndices(totalLines, maxShow, scrollOffset int) (startIdx, endIdx int) {
	maxScroll := totalLines - maxShow
	if maxScroll < 0 {
		maxScroll = 0
	}
	startIdx = scrollOffset
	if startIdx > maxScroll {
		startIdx = maxScroll
	}
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx = startIdx + maxShow
	if endIdx > totalLines {
		endIdx = totalLines
	}
	return startIdx, endIdx
}

func BoxWidth(termWidth, margin, minWidth int) int {
	w := termWidth - margin
	if w < minWidth {
		w = minWidth
	}
	return w
}

func CursorPrefix(selected bool) string {
	if selected {
		return "▸ "
	}
	return "  "
}

func FocusPrefix(focused bool) string {
	if focused {
		return "> "
	}
	return "  "
}

func RenderMenuItem(text string, selected bool) string {
	style := styles.NormalItem
	if selected {
		style = styles.SelectedItem
	}
	return CursorPrefix(selected) + style.Render(text)
}

type Toggle struct {
	Label       string
	Description string
	Value       bool
	Focused     bool
}

func (t *Toggle) View() string {
	checkbox := "[ ]"
	if t.Value {
		checkbox = "[✓]"
	}

	checkStyle := styles.SuccessText
	if !t.Value {
		checkStyle = styles.SubtleText
	}

	labelStyle := styles.NormalItem
	if t.Focused {
		labelStyle = styles.SelectedItem
	}

	line := fmt.Sprintf("  %s %s", checkStyle.Render(checkbox), labelStyle.Render(t.Label))

	if t.Description != "" {
		line += "\n" + styles.SubtleText.Render("      "+t.Description)
	}

	return line
}

func ConfirmDialog(message string) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(styles.WarningText.Render(message))
	b.WriteString("\n")
	b.WriteString(styles.SubtleText.Render("[y] confirm  [n] cancel"))

	return b.String()
}

