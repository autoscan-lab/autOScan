package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/felipetrejos/autoscan/internal/tui/styles"
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

	// Panel styling
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2).
		Width(h.width)

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.PrimaryGlow)

	b.WriteString(titleStyle.Render("Quick Reference"))
	b.WriteString("\n\n")

	// Description
	descStyle := lipgloss.NewStyle().Foreground(styles.Text)
	b.WriteString(descStyle.Render("Automated grading tool for"))
	b.WriteString("\n")
	b.WriteString(descStyle.Render("OS lab C submissions."))
	b.WriteString("\n\n")

	// Features
	featureStyle := lipgloss.NewStyle().Foreground(styles.Accent)
	features := []string{
		"• Batch compile with gcc",
		"• Detect banned functions",
		"• Run & test submissions",
		"• Multi-process execution",
		"• Real-time output streaming",
		"• Export to JSON/CSV",
	}
	for _, f := range features {
		b.WriteString(featureStyle.Render(f))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Shortcuts section
	shortcutHeaderStyle := lipgloss.NewStyle().
		Foreground(styles.Muted).
		Underline(true)

	b.WriteString(shortcutHeaderStyle.Render("Shortcuts"))
	b.WriteString("\n")

	for _, s := range h.shortcuts {
		key := styles.HelpKey.Render(fmt.Sprintf("%-8s", s.Key))
		desc := styles.HelpDesc.Render(s.Desc)
		b.WriteString(fmt.Sprintf("%s %s\n", key, desc))
	}

	b.WriteString("\n")

	// Status section
	statusStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	b.WriteString(statusStyle.Render("───────────────"))
	b.WriteString("\n")

	// Version
	versionLabel := lipgloss.NewStyle().Foreground(styles.Muted).Render("Version: ")
	versionValue := lipgloss.NewStyle().Foreground(styles.Success).Render(h.version)
	b.WriteString(versionLabel + versionValue)
	b.WriteString("\n")

	// Policy count
	if h.policyCount > 0 {
		countLabel := lipgloss.NewStyle().Foreground(styles.Muted).Render("Policies: ")
		countValue := lipgloss.NewStyle().Foreground(styles.Accent).Render(fmt.Sprintf("%d loaded", h.policyCount))
		b.WriteString(countLabel + countValue)
	} else {
		b.WriteString(styles.WarningText.Render("No policies found"))
	}

	return panelStyle.Render(b.String())
}

func RenderHelpBar(items []HelpItem) string {
	var parts []string
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s %s",
			styles.HelpKey.Render(item.Key),
			styles.HelpDesc.Render(item.Desc)))
	}
	return "  " + strings.Join(parts, "  •  ")
}

