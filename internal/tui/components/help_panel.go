package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/felipetrejos/autoscan/internal/tui/styles"
)

// HelpItem represents a single keyboard shortcut
type HelpItem struct {
	Key  string
	Desc string
}

// HelpPanel displays tips and shortcuts on the home screen
type HelpPanel struct {
	width        int
	version      string
	policyCount  int
	shortcuts    []HelpItem
}

// NewHelpPanel creates a new help panel
func NewHelpPanel(width int, version string) HelpPanel {
	return HelpPanel{
		width:   width,
		version: version,
		shortcuts: []HelpItem{
			{"↑/↓", "Navigate"},
			{"Enter", "Select"},
			{"1-5", "Quick jump"},
			{"q", "Quit"},
		},
	}
}

// SetPolicyCount updates the policy count display
func (h *HelpPanel) SetPolicyCount(count int) {
	h.policyCount = count
}

// SetWidth updates the panel width
func (h *HelpPanel) SetWidth(width int) {
	h.width = width
}

// View renders the help panel
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
		"• Export to MD/JSON/CSV",
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

// RenderHelpBar creates a simple inline help bar for any view
func RenderHelpBar(items []HelpItem) string {
	var parts []string
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s %s",
			styles.HelpKey.Render(item.Key),
			styles.HelpDesc.Render(item.Desc)))
	}
	return "  " + strings.Join(parts, "  •  ")
}

// ContextualHelp provides context-aware tips based on menu selection
type ContextualHelp struct {
	tips map[int][]string
}

// NewContextualHelp creates contextual help tips
func NewContextualHelp() ContextualHelp {
	return ContextualHelp{
		tips: map[int][]string{
			0: {"Select a policy to start grading", "Press Enter to continue"},
			1: {"Create, edit, or delete policies", "Manage banned function list"},
			2: {"Configure display options", "Settings are saved automatically"},
			3: {"Remove autOScan and all configs", "This action is irreversible"},
			4: {"Exit the application", "Progress is auto-saved"},
		},
	}
}

// GetTips returns tips for a given menu index
func (c *ContextualHelp) GetTips(menuIndex int) []string {
	if tips, ok := c.tips[menuIndex]; ok {
		return tips
	}
	return []string{}
}
