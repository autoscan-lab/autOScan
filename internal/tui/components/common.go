package components

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/feli05/autoscan/internal/tui/styles"
)

const tabWidth = 8

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

func SanitizeDisplay(s string) string {
	if s == "" {
		return s
	}
	s = ansiRegexp.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, s)
	return s
}

func expandTabs(s string, width int) string {
	if !strings.Contains(s, "\t") {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))
	col := 0
	for _, r := range s {
		if r == '\t' {
			spaces := width - (col % width)
			if spaces == 0 {
				spaces = width
			}
			b.WriteString(strings.Repeat(" ", spaces))
			col += spaces
			continue
		}
		b.WriteRune(r)
		col += lipgloss.Width(string(r))
	}
	return b.String()
}

// TruncateToWidth truncates a string to fit within maxWidth display columns.
// Uses lipgloss.Width for accurate display width calculation.
func TruncateToWidth(s string, maxWidth int) string {
	s = SanitizeDisplay(s)
	s = expandTabs(s, tabWidth)
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	// Truncate rune by rune until it fits
	runes := []rune(s)
	for i := len(runes); i > 0; i-- {
		truncated := string(runes[:i]) + "..."
		if lipgloss.Width(truncated) <= maxWidth {
			return truncated
		}
	}
	return "..."
}

func WrapLines(text string, width int) []string {
	if text == "" {
		return nil
	}
	if width <= 0 {
		return []string{""}
	}
	var wrapped []string
	for _, line := range strings.Split(text, "\n") {
		line = expandTabs(SanitizeDisplay(line), tabWidth)
		if lipgloss.Width(line) <= width {
			wrapped = append(wrapped, line)
		} else {
			var current strings.Builder
			currentWidth := 0
			for _, r := range line {
				rw := lipgloss.Width(string(r))
				if currentWidth+rw > width {
					wrapped = append(wrapped, current.String())
					current.Reset()
					currentWidth = 0
				}
				current.WriteRune(r)
				currentWidth += rw
			}
			if current.Len() > 0 || len(line) == 0 {
				wrapped = append(wrapped, current.String())
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

type NumberSetting struct {
	Label       string
	Value       string
	Description []string
	Focused     bool
}

func (ns NumberSetting) View() string {
	var b strings.Builder

	line := fmt.Sprintf("  %s: %s", ns.Label, ns.Value)
	if ns.Focused {
		b.WriteString(styles.SelectedItem.Render(line))
	} else {
		b.WriteString(styles.NormalItem.Render(line))
	}

	for _, desc := range ns.Description {
		b.WriteString("\n")
		b.WriteString(styles.SubtleText.Render("      " + desc))
	}

	return b.String()
}
