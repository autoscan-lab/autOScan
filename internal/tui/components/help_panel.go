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
	var b strings.Builder

	panelStyle := PrimaryBoxStyle().Width(h.width)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(PrimaryGlow)

	b.WriteString(titleStyle.Render("Quick Reference"))
	b.WriteString("\n\n")

	descStyle := lipgloss.NewStyle().Foreground(Text)
	b.WriteString(descStyle.Render("Automated grading tool for"))
	b.WriteString("\n")
	b.WriteString(descStyle.Render("OS lab C submissions."))
	b.WriteString("\n\n")

	featureStyle := lipgloss.NewStyle().Foreground(Accent)
	features := []string{
		"• Batch compile with gcc",
		"• Detect banned functions",
		"• Detect AI-like code patterns",
		"• Run & test submissions",
		"• Multi-process execution",
		"• Similarity and AI detail views",
		"• Real-time output streaming",
		"• Export to JSON/CSV",
	}
	for _, f := range features {
		b.WriteString(featureStyle.Render(f))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	shortcutHeaderStyle := lipgloss.NewStyle().
		Foreground(Muted).
		Underline(true)

	b.WriteString(shortcutHeaderStyle.Render("Shortcuts"))
	b.WriteString("\n")

	for _, s := range h.shortcuts {
		key := HelpKey.Render(fmt.Sprintf("%-8s", s.Key))
		desc := HelpDesc.Render(s.Desc)
		b.WriteString(fmt.Sprintf("%s %s\n", key, desc))
	}

	b.WriteString("\n")

	statusStyle := lipgloss.NewStyle().Foreground(Muted)
	b.WriteString(statusStyle.Render("───────────────"))
	b.WriteString("\n")

	versionLabel := lipgloss.NewStyle().Foreground(Muted).Render("Version: ")
	versionValue := lipgloss.NewStyle().Foreground(Success).Render(h.version)
	b.WriteString(versionLabel + versionValue)
	b.WriteString("\n")

	if h.policyCount > 0 {
		countLabel := lipgloss.NewStyle().Foreground(Muted).Render("Policies: ")
		countValue := lipgloss.NewStyle().Foreground(Accent).Render(fmt.Sprintf("%d loaded", h.policyCount))
		b.WriteString(countLabel + countValue)
	} else {
		b.WriteString(WarningText.Render("No policies found"))
	}

	return panelStyle.Render(b.String())
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
