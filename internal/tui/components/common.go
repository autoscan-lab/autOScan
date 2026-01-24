package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

type ListItem struct {
	ID, Title, Description, Status string
	Extra                          interface{}
}

type List struct {
	Items        []ListItem
	Cursor       int
	ScrollOffset int
	VisibleRows  int
	Width        int
	ShowIndex    bool
}

func NewList(visibleRows, width int) List {
	return List{
		Items:       []ListItem{},
		VisibleRows: visibleRows,
		Width:       width,
	}
}

func (l *List) SetItems(items []ListItem) {
	l.Items = items
	l.Cursor = 0
	l.ScrollOffset = 0
}

func (l *List) MoveUp() {
	if l.Cursor > 0 {
		l.Cursor--
		if l.Cursor < l.ScrollOffset {
			l.ScrollOffset--
		}
	}
}

func (l *List) MoveDown() {
	if l.Cursor < len(l.Items)-1 {
		l.Cursor++
		if l.Cursor >= l.ScrollOffset+l.VisibleRows {
			l.ScrollOffset++
		}
	}
}

func (l *List) Selected() *ListItem {
	if l.Cursor >= 0 && l.Cursor < len(l.Items) {
		return &l.Items[l.Cursor]
	}
	return nil
}

func (l *List) View() string {
	if len(l.Items) == 0 {
		return styles.SubtleText.Render("  (no items)")
	}

	var b strings.Builder

	endIdx := l.ScrollOffset + l.VisibleRows
	if endIdx > len(l.Items) {
		endIdx = len(l.Items)
	}

	for i := l.ScrollOffset; i < endIdx; i++ {
		item := l.Items[i]
		cursor := "  "
		style := styles.NormalItem

		if i == l.Cursor {
			cursor = "▸ "
			style = styles.SelectedItem
		}

		// Build line
		line := cursor
		if l.ShowIndex {
			line += fmt.Sprintf("[%d] ", i+1)
		}
		line += item.Title

		b.WriteString(style.Render(line))
		b.WriteString("\n")

		// Show description for selected item
		if i == l.Cursor && item.Description != "" {
			b.WriteString(styles.SubtleText.Render("    " + item.Description))
			b.WriteString("\n")
		}
	}

	// Scroll indicator
	if len(l.Items) > l.VisibleRows {
		b.WriteString("\n")
		b.WriteString(styles.SubtleText.Render(fmt.Sprintf("  (%d-%d of %d)",
			l.ScrollOffset+1, endIdx, len(l.Items))))
	}

	return b.String()
}

func StatusBadge(status string) string {
	switch status {
	case "clean", "ok", "pass", "success":
		return styles.StatusClean.Render("✓ " + status)
	case "warning", "warn", "banned":
		return styles.StatusWarning.Render("⚠ " + status)
	case "error", "fail", "failed":
		return styles.StatusError.Render("✗ " + status)
	default:
		return styles.StatusMuted.Render("○ " + status)
	}
}

func StatusIcon(status string) string {
	switch status {
	case "clean", "ok", "pass", "success":
		return styles.SuccessText.Render("✓")
	case "warning", "warn", "banned":
		return styles.WarningText.Render("⚠")
	case "error", "fail", "failed":
		return styles.ErrorText.Render("✗")
	case "running", "loading":
		return styles.PrimaryText.Render("◉")
	default:
		return styles.SubtleText.Render("○")
	}
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

func ProgressBar(current, total, width int) string {
	if total == 0 {
		return ""
	}

	percent := float64(current) / float64(total)
	filled := int(percent * float64(width))
	if filled > width {
		filled = width
	}

	filledStyle := lipgloss.NewStyle().Foreground(styles.Primary)
	emptyStyle := lipgloss.NewStyle().Foreground(styles.Muted)

	bar := filledStyle.Render(strings.Repeat("█", filled))
	bar += emptyStyle.Render(strings.Repeat("░", width-filled))

	return fmt.Sprintf("[%s] %d/%d", bar, current, total)
}

func SectionHeader(title string, width int) string {
	titleStyled := styles.HeaderStyle.Render(title)
	lineWidth := width - lipgloss.Width(title) - 4
	if lineWidth < 0 {
		lineWidth = 0
	}
	line := strings.Repeat("─", lineWidth)
	return titleStyled + " " + styles.SubtleText.Render(line)
}

func ConfirmDialog(message string) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(styles.WarningText.Render(message))
	b.WriteString("\n")
	b.WriteString(styles.SubtleText.Render("[y] confirm  [n] cancel"))

	return b.String()
}

func KeyValue(key, value string) string {
	return styles.SubtleText.Render(key+": ") + styles.NormalItem.Render(value)
}

func KeyValueHighlight(key, value string) string {
	return styles.SubtleText.Render(key+": ") + styles.Highlight.Render(value)
}
