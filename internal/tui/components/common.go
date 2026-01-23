package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/felipetrejos/autoscan/internal/tui/styles"
)

// ─────────────────────────────────────────────────────────────────────────────
// List Component
// ─────────────────────────────────────────────────────────────────────────────

// ListItem represents an item in a selectable list
type ListItem struct {
	ID          string
	Title       string
	Description string
	Status      string
	Extra       interface{}
}

// List is a reusable scrollable list component
type List struct {
	Items        []ListItem
	Cursor       int
	ScrollOffset int
	VisibleRows  int
	Width        int
	ShowIndex    bool
}

// NewList creates a new list component
func NewList(visibleRows, width int) List {
	return List{
		Items:       []ListItem{},
		VisibleRows: visibleRows,
		Width:       width,
	}
}

// SetItems updates the list items
func (l *List) SetItems(items []ListItem) {
	l.Items = items
	l.Cursor = 0
	l.ScrollOffset = 0
}

// MoveUp moves cursor up
func (l *List) MoveUp() {
	if l.Cursor > 0 {
		l.Cursor--
		if l.Cursor < l.ScrollOffset {
			l.ScrollOffset--
		}
	}
}

// MoveDown moves cursor down
func (l *List) MoveDown() {
	if l.Cursor < len(l.Items)-1 {
		l.Cursor++
		if l.Cursor >= l.ScrollOffset+l.VisibleRows {
			l.ScrollOffset++
		}
	}
}

// Selected returns the currently selected item
func (l *List) Selected() *ListItem {
	if l.Cursor >= 0 && l.Cursor < len(l.Items) {
		return &l.Items[l.Cursor]
	}
	return nil
}

// View renders the list
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

// ─────────────────────────────────────────────────────────────────────────────
// Status Badge - Colored status indicators
// ─────────────────────────────────────────────────────────────────────────────

// StatusBadge returns a styled status indicator
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

// StatusIcon returns just the icon for a status
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

// ─────────────────────────────────────────────────────────────────────────────
// Toggle Component - Boolean toggle
// ─────────────────────────────────────────────────────────────────────────────

// Toggle represents a boolean toggle setting
type Toggle struct {
	Label       string
	Description string
	Value       bool
	Focused     bool
}

// View renders the toggle
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

// ─────────────────────────────────────────────────────────────────────────────
// Progress Bar
// ─────────────────────────────────────────────────────────────────────────────

// ProgressBar renders a progress indicator
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

// ─────────────────────────────────────────────────────────────────────────────
// Section Header
// ─────────────────────────────────────────────────────────────────────────────

// SectionHeader creates a styled section header
func SectionHeader(title string, width int) string {
	titleStyled := styles.HeaderStyle.Render(title)
	lineWidth := width - lipgloss.Width(title) - 4
	if lineWidth < 0 {
		lineWidth = 0
	}
	line := strings.Repeat("─", lineWidth)
	return titleStyled + " " + styles.SubtleText.Render(line)
}

// ─────────────────────────────────────────────────────────────────────────────
// Confirm Dialog
// ─────────────────────────────────────────────────────────────────────────────

// ConfirmDialog renders a confirmation prompt
func ConfirmDialog(message string) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(styles.WarningText.Render(message))
	b.WriteString("\n")
	b.WriteString(styles.SubtleText.Render("[y] confirm  [n] cancel"))

	return b.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// Key Value Display
// ─────────────────────────────────────────────────────────────────────────────

// KeyValue renders a key-value pair
func KeyValue(key, value string) string {
	return styles.SubtleText.Render(key+": ") + styles.NormalItem.Render(value)
}

// KeyValueHighlight renders a key-value pair with the value highlighted
func KeyValueHighlight(key, value string) string {
	return styles.SubtleText.Render(key+": ") + styles.Highlight.Render(value)
}
