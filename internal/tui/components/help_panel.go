package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type HelpItem struct {
	Key, Desc string
}

type HelpPanel struct {
	width       int
	version     string
	policyCount int
	shortcuts   []HelpItem
}

func NewHelpPanel(width int, version string) HelpPanel {
	return HelpPanel{
		width:   width,
		version: version,
		shortcuts: []HelpItem{
			{"↑/↓", "Navigate"},
			{"Enter", "Select"},
			{"1-4", "Quick jump"},
			{"q", "Quit"},
		},
	}
}

func (h *HelpPanel) SetPolicyCount(count int) { h.policyCount = count }
func (h *HelpPanel) SetWidth(width int)       { h.width = width }

func (h *HelpPanel) View() string {
	panelStyle := PrimaryBoxStyle().Width(h.width)
	contentWidth := max(26, h.width-6)
	gap := 3
	if contentWidth < 48 {
		gap = 2
	}
	innerWidth := contentWidth - (gap * 2)
	if innerWidth < 30 {
		innerWidth = contentWidth
		gap = 1
	}

	leftW := innerWidth * 44 / 100
	midW := innerWidth * 31 / 100
	rightW := innerWidth - leftW - midW

	if leftW < 16 || midW < 12 || rightW < 10 {
		leftW = innerWidth * 45 / 100
		midW = innerWidth * 30 / 100
		rightW = innerWidth - leftW - midW
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(PrimaryGlow)
	descStyle := lipgloss.NewStyle().Foreground(Text)
	sectionStyle := lipgloss.NewStyle().Foreground(Muted).Underline(true)

	var left strings.Builder
	left.WriteString(titleStyle.Render("Quick Reference"))
	left.WriteString("\n")
	for _, line := range wrapWords("Fast checks for C lab submissions.", leftW) {
		left.WriteString(descStyle.Render(line))
		left.WriteString("\n")
	}

	var middle strings.Builder
	middle.WriteString(sectionStyle.Render("Shortcuts"))
	middle.WriteString("\n")
	for _, s := range h.shortcuts {
		key := HelpKey.Render(fmt.Sprintf("%-5s", s.Key))
		desc := HelpDesc.Render(s.Desc)
		middle.WriteString(fmt.Sprintf("%s %s\n", key, desc))
	}

	var right strings.Builder
	right.WriteString(sectionStyle.Render("Status"))
	right.WriteString("\n")
	versionPart := lipgloss.NewStyle().Foreground(Muted).Render("Version: ") +
		lipgloss.NewStyle().Foreground(Success).Render(h.version)
	right.WriteString(versionPart)
	right.WriteString("\n")

	policyPart := WarningText.Render("No policies found")
	if h.policyCount > 0 {
		policyPart = lipgloss.NewStyle().Foreground(Muted).Render("Policies: ") +
			lipgloss.NewStyle().Foreground(Accent).Render(fmt.Sprintf("%d loaded", h.policyCount))
	}
	right.WriteString(policyPart)

	leftCol := lipgloss.NewStyle().Width(leftW).MaxWidth(leftW).Render(strings.TrimRight(left.String(), "\n"))
	middleCol := lipgloss.NewStyle().Width(midW).MaxWidth(midW).Render(strings.TrimRight(middle.String(), "\n"))
	rightCol := lipgloss.NewStyle().Width(rightW).MaxWidth(rightW).Render(strings.TrimRight(right.String(), "\n"))

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftCol,
		strings.Repeat(" ", gap),
		middleCol,
		strings.Repeat(" ", gap),
		rightCol,
	)
	return panelStyle.Render(content)
}

func RenderHelpBar(items []HelpItem) string {
	var parts []string
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s %s",
			HelpKey.Render(item.Key),
			HelpDesc.Render(item.Desc)))
	}
	return "  " + strings.Join(parts, "  •  ")
}

func wrapWords(text string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	words := strings.Fields(SanitizeDisplay(text))
	if len(words) == 0 {
		return []string{""}
	}

	lines := make([]string, 0, len(words))
	current := words[0]
	for _, w := range words[1:] {
		candidate := current + " " + w
		if lipgloss.Width(candidate) <= width {
			current = candidate
			continue
		}
		lines = append(lines, current)
		if lipgloss.Width(w) > width {
			lines = append(lines, WrapLines(w, width)...)
			current = ""
			continue
		}
		current = w
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
