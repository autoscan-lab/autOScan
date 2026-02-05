package components

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const tabWidth = 8

var ansiRegexp = regexp.MustCompile(`\x1b\\[[0-9;?]*[A-Za-z]`)

type Toggle struct {
	Label       string
	Description string
	Value       bool
	Focused     bool
}

type NumberSetting struct {
	Label       string
	Value       string
	Description []string
	Focused     bool
}

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
	style := NormalItem
	if selected {
		style = SelectedItem
	}
	return CursorPrefix(selected) + style.Render(text)
}

func (t *Toggle) View() string {
	checkbox := "[ ]"
	if t.Value {
		checkbox = "[✓]"
	}

	checkStyle := SuccessText
	if !t.Value {
		checkStyle = SubtleText
	}

	labelStyle := NormalItem
	if t.Focused {
		labelStyle = SelectedItem
	}

	line := fmt.Sprintf("  %s %s", checkStyle.Render(checkbox), labelStyle.Render(t.Label))

	if t.Description != "" {
		line += "\n" + SubtleText.Render("      "+t.Description)
	}

	return line
}

func ConfirmDialog(message string) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(WarningText.Render(message))
	b.WriteString("\n")
	b.WriteString(SubtleText.Render("[y] confirm  [n] cancel"))

	return b.String()
}

func (ns NumberSetting) View() string {
	var b strings.Builder

	line := fmt.Sprintf("  %s: %s", ns.Label, ns.Value)
	if ns.Focused {
		b.WriteString(SelectedItem.Render(line))
	} else {
		b.WriteString(NormalItem.Render(line))
	}

	for _, desc := range ns.Description {
		b.WriteString("\n")
		b.WriteString(SubtleText.Render("      " + desc))
	}

	return b.String()
}

func RenderHeader(title string) string {
	return HeaderStyle.Render(title) + "\n\n"
}

func TruncatePathToFilename(s string) string {
	if strings.Contains(s, "/") && !strings.HasPrefix(s, "-") {
		return filepath.Base(s)
	}
	return s
}

func TruncatePathsInText(text string) string {
	result := text
	parts := strings.Split(result, "/")
	if len(parts) > 1 {
		result = truncateAbsolutePaths(result)
	}
	return result
}

func truncateAbsolutePaths(text string) string {
	var result strings.Builder
	i := 0
	for i < len(text) {
		if text[i] == '/' && i+1 < len(text) && (text[i+1] == 'U' || text[i+1] == 'h' || text[i+1] == 'v' || text[i+1] == 't') {
			pathEnd := i + 1
			for pathEnd < len(text) && text[pathEnd] != ' ' && text[pathEnd] != ':' && text[pathEnd] != '\n' && text[pathEnd] != ')' && text[pathEnd] != '(' {
				pathEnd++
			}
			if pathEnd > i+1 {
				pathStr := text[i:pathEnd]
				if strings.Count(pathStr, "/") > 2 {
					filename := filepath.Base(pathStr)
					result.WriteString(filename)
					i = pathEnd
					continue
				}
			}
		}
		result.WriteByte(text[i])
		i++
	}
	return result.String()
}
