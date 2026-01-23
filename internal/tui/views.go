package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/felipetrejos/autoscan/internal/domain"
	"github.com/felipetrejos/autoscan/internal/export"
	"github.com/felipetrejos/autoscan/internal/tui/components"
	"github.com/felipetrejos/autoscan/internal/tui/styles"
)

// ASCII Art Logo - Compact version for new layout
const logo = `
 █████╗ ██╗   ██╗████████╗ ██████╗ ███████╗ ██████╗ █████╗ ███╗   ██╗
██╔══██╗██║   ██║╚══██╔══╝██╔═══██╗██╔════╝██╔════╝██╔══██╗████╗  ██║
███████║██║   ██║   ██║   ██║   ██║███████╗██║     ███████║██╔██╗ ██║
██╔══██║██║   ██║   ██║   ██║   ██║╚════██║██║     ██╔══██║██║╚██╗██║
██║  ██║╚██████╔╝   ██║   ╚██████╔╝███████║╚██████╗██║  ██║██║ ╚████║
╚═╝  ╚═╝ ╚═════╝    ╚═╝    ╚═════╝ ╚══════╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═══╝`

const tagline = "OS Lab Submission Grader"

// ─────────────────────────────────────────────────────────────────────────────
// Main View Router
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	var content string

	switch m.currentView {
	case ViewHome:
		content = m.renderHome()
	case ViewPolicySelect:
		content = m.renderPolicySelect()
	case ViewPolicyManage:
		content = m.renderPolicyManage()
	case ViewPolicyEditor:
		content = m.policyEditor.View()
	case ViewBannedEditor:
		content = m.renderBannedEditor()
	case ViewSettings:
		content = m.renderSettings()
	case ViewDirectoryInput:
		content = m.renderDirectoryInput()
	case ViewSubmissions:
		content = m.renderSubmissions()
	case ViewDetails:
		content = m.renderDetails()
	case ViewExport:
		content = m.renderExport()
	default:
		content = m.renderHome()
	}

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Left,
		lipgloss.Top,
		content,
	)
}

// ─────────────────────────────────────────────────────────────────────────────
// Home View
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) renderHome() string {
	var b strings.Builder

	// Calculate layout dimensions
	contentWidth := min(100, m.width)
	menuWidth := min(50, contentWidth*2/3)
	helpPanelWidth := min(35, contentWidth-menuWidth-2)

	// ─────────────────────────────────────────────────────────────────────────
	// Top Section: Logo + Animation side by side
	// ─────────────────────────────────────────────────────────────────────────

	// Logo
	logoStyled := styles.LogoStyle.Render(logo)
	taglineStyled := styles.SubtleText.Render("     " + tagline)

	// Animation (positioned to the right)
	animation := m.eyeAnimation.View()
	animationBox := lipgloss.NewStyle().
		Width(20).
		Align(lipgloss.Center).
		Render(animation)

	// Place logo and animation in the same horizontal space
	logoWithTagline := logoStyled + "\n" + taglineStyled

	topSection := lipgloss.JoinHorizontal(
		lipgloss.Top,
		logoWithTagline,
		lipgloss.NewStyle().PaddingLeft(4).Render(animationBox),
	)

	b.WriteString(topSection)
	b.WriteString("\n\n")

	// ─────────────────────────────────────────────────────────────────────────
	// Bottom Section: Menu + Help Panel
	// ─────────────────────────────────────────────────────────────────────────

	// Menu box
	menuBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 3).
		Width(menuWidth)

	var menu strings.Builder
	menuItems := []struct {
		key  string
		desc string
		item MenuItem
	}{
		{"1", "Run Grader", MenuRunGrader},
		{"2", "Manage Policies", MenuManagePolicies},
		{"3", "Settings", MenuSettings},
		{"4", "Uninstall", MenuUninstall},
		{"q", "Quit", MenuQuit},
	}

	for _, mi := range menuItems {
		cursor := "  "
		style := styles.NormalItem
		if mi.item == m.menuItem {
			cursor = "▸ "
			style = styles.SelectedItem
		}
		keyStyle := styles.HelpKey.Render(fmt.Sprintf("[%s]", mi.key))
		menu.WriteString(fmt.Sprintf("%s%s %s\n", cursor, keyStyle, style.Render(mi.desc)))
	}

	// Show uninstall confirmation if active
	if m.confirmDelete && m.menuItem == MenuUninstall {
		menu.WriteString("\n")
		menu.WriteString(components.ConfirmDialog("Remove autoscan and all configs?"))
	}

	menuRendered := menuBox.Render(menu.String())

	// Help panel
	m.helpPanel.SetWidth(helpPanelWidth)
	m.helpPanel.SetPolicyCount(len(m.policies))
	helpRendered := m.helpPanel.View()

	// Combine menu and help panel horizontally
	bottomSection := lipgloss.JoinHorizontal(
		lipgloss.Top,
		menuRendered,
		lipgloss.NewStyle().MarginLeft(2).Render(helpRendered),
	)

	b.WriteString(bottomSection)

	// ─────────────────────────────────────────────────────────────────────────
	// Footer
	// ─────────────────────────────────────────────────────────────────────────

	b.WriteString("\n\n")
	b.WriteString(styles.SubtleText.Render("  Use ↑/↓ to navigate, Enter to select"))

	return b.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// Policy Select View
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) renderPolicySelect() string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("Select a Policy"))
	b.WriteString("\n\n")

	boxWidth := min(65, m.width-4)

	if len(m.policies) == 0 {
		box := styles.WarningBoxStyle(boxWidth)
		content := styles.WarningText.Render("No policies found!") + "\n\n" +
			styles.SubtleText.Render("Create a policy via Manage Policies or edit ~/.config/autoscan/")
		b.WriteString(box.Render(content))
	} else {
		// Main selection box with primary border
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Primary).
			Padding(1, 2).
			Width(boxWidth)

		var list strings.Builder

		// Header with count
		list.WriteString(styles.SubtleText.Render(fmt.Sprintf("Available policies: %d", len(m.policies))))
		list.WriteString("\n\n")

		for i, p := range m.policies {
			cursor := "  "
			style := styles.NormalItem
			if i == m.selectedPolicy {
				cursor = "▸ "
				style = styles.SelectedItem
			}

			list.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(p.Name)))
		}

		b.WriteString(box.Render(list.String()))

		// Separate details panel for selected policy
		if m.selectedPolicy < len(m.policies) {
			b.WriteString("\n\n")

			detailBox := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(styles.Muted).
				Padding(1, 2).
				Width(boxWidth)

			var details strings.Builder
			p := m.policies[m.selectedPolicy]

			details.WriteString(styles.Highlight.Render("Policy Details"))
			details.WriteString("\n\n")

			// Policy name
			details.WriteString(styles.SubtleText.Render("  Name:     "))
			details.WriteString(p.Name)
			details.WriteString("\n")

			// File path
			relPath, _ := filepath.Rel(".", p.FilePath)
			details.WriteString(styles.SubtleText.Render("  File:     "))
			details.WriteString(filepath.Base(relPath))
			details.WriteString("\n")

			// Compiler flags
			details.WriteString(styles.SubtleText.Render("  Flags:    "))
			if len(p.Compile.Flags) > 0 {
				details.WriteString(strings.Join(p.Compile.Flags, " "))
			} else {
				details.WriteString(styles.SubtleText.Render("(default)"))
			}
			details.WriteString("\n")

			// Output binary
			details.WriteString(styles.SubtleText.Render("  Output:   "))
			if p.Compile.Output != "" {
				details.WriteString(p.Compile.Output)
			} else {
				details.WriteString("a.out")
			}
			details.WriteString("\n")

			// Required files
			if len(p.RequiredFiles) > 0 {
				details.WriteString(styles.SubtleText.Render("  Required: "))
				details.WriteString(strings.Join(p.RequiredFiles, ", "))
				details.WriteString("\n")
			}

			b.WriteString(detailBox.Render(details.String()))
		}
	}

	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "select"},
		{Key: "esc", Desc: "back"},
	}))

	return b.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// Policy Manage View
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) renderPolicyManage() string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("Manage Policies"))
	b.WriteString("\n\n")

	boxWidth := min(60, m.width-4)

	// ─────────────────────────────────────────────────────────────────────────
	// Section 1: Configuration
	// ─────────────────────────────────────────────────────────────────────────

	configBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(1, 2).
		Width(boxWidth)

	var configSection strings.Builder
	configSection.WriteString(styles.SubtleText.Render("Configuration"))
	configSection.WriteString("\n\n")

	// Edit banned functions option
	if m.policyManageCursor == -1 {
		configSection.WriteString("▸ ")
		configSection.WriteString(styles.SelectedItem.Render("Edit Banned Functions"))
	} else {
		configSection.WriteString("  ")
		configSection.WriteString(styles.NormalItem.Render("Edit Banned Functions"))
	}
	configSection.WriteString("\n")
	configSection.WriteString(styles.SubtleText.Render("    Global list of prohibited function calls"))

	b.WriteString(configBox.Render(configSection.String()))
	b.WriteString("\n\n")

	// ─────────────────────────────────────────────────────────────────────────
	// Section 2: Policies
	// ─────────────────────────────────────────────────────────────────────────

	policyBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2).
		Width(boxWidth)

	var policySection strings.Builder
	policySection.WriteString(styles.PrimaryText.Render(fmt.Sprintf("Policies (%d)", len(m.policies))))
	policySection.WriteString("\n\n")

	// Create new policy option
	if m.policyManageCursor == 0 {
		policySection.WriteString("▸ ")
		policySection.WriteString(styles.SelectedItem.Render("+ Create New Policy"))
	} else {
		policySection.WriteString("  ")
		policySection.WriteString(styles.NormalItem.Render("+ Create New Policy"))
	}
	policySection.WriteString("\n")

	// Separator before existing policies
	if len(m.policies) > 0 {
		policySection.WriteString("\n")
	}

	// Existing policies
	for i, p := range m.policies {
		if m.policyManageCursor == i+1 {
			policySection.WriteString("▸ ")
			policySection.WriteString(styles.SelectedItem.Render(p.Name))
		} else {
			policySection.WriteString("  ")
			policySection.WriteString(styles.NormalItem.Render(p.Name))
		}
		policySection.WriteString("\n")
	}

	// Show selected policy details at bottom
	if m.policyManageCursor > 0 && m.policyManageCursor <= len(m.policies) {
		p := m.policies[m.policyManageCursor-1]
		policySection.WriteString("\n")
		policySection.WriteString(styles.SubtleText.Render(fmt.Sprintf("  File: %s", filepath.Base(p.FilePath))))
		if len(p.Compile.Flags) > 0 {
			policySection.WriteString("\n")
			policySection.WriteString(styles.SubtleText.Render(fmt.Sprintf("  Flags: %s", strings.Join(p.Compile.Flags, " "))))
		}
	}

	// Confirm delete dialog
	if m.confirmDelete && m.policyManageCursor > 0 {
		policySection.WriteString("\n")
		policySection.WriteString(components.ConfirmDialog("Delete this policy?"))
	}

	b.WriteString(policyBox.Render(policySection.String()))

	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "select"},
		{Key: "d", Desc: "delete"},
		{Key: "esc", Desc: "back"},
	}))

	return b.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// Settings View (Updated with KeepBinaries)
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) renderSettings() string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("Settings"))
	b.WriteString("\n\n")

	box := styles.BoxStyle(min(55, m.width-4))

	var content strings.Builder
	content.WriteString(styles.SubtleText.Render("Display Options"))
	content.WriteString("\n\n")

	// Short names toggle
	toggle1 := components.Toggle{
		Label:       "Short Names",
		Description: "Truncate folder names at first underscore",
		Value:       m.settings.ShortNames,
		Focused:     m.settingsCursor == 0,
	}
	content.WriteString(toggle1.View())
	content.WriteString("\n\n")

	// Keep binaries toggle
	toggle2 := components.Toggle{
		Label:       "Keep Binaries",
		Description: "Keep compiled binaries after grading",
		Value:       m.settings.KeepBinaries,
		Focused:     m.settingsCursor == 1,
	}
	content.WriteString(toggle2.View())

	b.WriteString(box.Render(content.String()))

	b.WriteString("\n\n")
	b.WriteString(styles.SubtleText.Render("  Config: ~/.config/autoscan/settings.yaml"))

	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{"↑/↓", "navigate"},
		{"space/enter", "toggle"},
		{"esc", "back"},
	}))

	return b.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// Directory Input View
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) renderDirectoryInput() string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("Select Directory"))
	b.WriteString("\n\n")

	box := styles.BoxStyle(min(60, m.width-4))

	var content strings.Builder
	content.WriteString(styles.SubtleText.Render("Navigate to submissions folder"))
	content.WriteString("\n\n")
	content.WriteString(m.folderBrowser.View())

	b.WriteString(box.Render(content.String()))

	if m.inputError != "" {
		b.WriteString("\n")
		b.WriteString(styles.ErrorText.Render("  " + m.inputError))
	}

	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{"↑/↓", "navigate"},
		{"enter", "select/open"},
		{"←/backspace", "parent dir"},
		{"esc", "cancel"},
	}))

	return b.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// Submissions View
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) renderSubmissions() string {
	var b strings.Builder

	// Header with policy name
	policyName := "Unknown Policy"
	if m.selectedPolicy < len(m.policies) {
		policyName = m.policies[m.selectedPolicy].Name
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		Padding(0, 2)

	b.WriteString("\n")
	b.WriteString(header.Render(policyName))
	b.WriteString("\n")
	b.WriteString(styles.SubtleText.Render(fmt.Sprintf("  %s", m.root)))
	b.WriteString("\n")

	if m.runError != "" {
		b.WriteString("\n")
		errorBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Error).
			Padding(1, 2)
		b.WriteString(errorBox.Render(styles.ErrorText.Render("Error: " + m.runError)))
		b.WriteString("\n")
	} else if m.isRunning {
		b.WriteString(fmt.Sprintf("\n  %s Scanning and compiling...\n", m.spinner.View()))
	} else if m.report != nil {
		// Summary stats
		statsBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Muted).
			Padding(0, 2).
			MarginTop(1)

		stats := fmt.Sprintf(
			"Pass: %d  Fail: %d  Banned: %d  Time: %dms  Filter: %s",
			m.report.Summary.CompilePass,
			m.report.Summary.CompileFail,
			m.report.Summary.SubmissionsWithBanned,
			m.report.Summary.DurationMs,
			m.filter.String(),
		)
		b.WriteString(statsBox.Render(stats))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Table
	tableBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(0, 1)

	var table strings.Builder

	// Column widths for proper alignment
	const (
		colStatus     = 5  // [OK], [!], [X], [~] + space
		colSubmission = 34 // Wide enough for longer names
		colCompile    = 9
		colBanned     = 8
		colGrade      = 6
	)

	// Table header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary)

	// Header: "  " cursor + status + submission + compile + banned + grade
	table.WriteString(headerStyle.Render(fmt.Sprintf("  %-*s %-*s  %-*s  %-*s  %-*s",
		colStatus, "",
		colSubmission, "Submission",
		colCompile, "Compile",
		colBanned, "Banned",
		colGrade, "Grade")))
	table.WriteString("\n")
	table.WriteString(strings.Repeat("─", 2+colStatus+1+colSubmission+2+colCompile+2+colBanned+2+colGrade))
	table.WriteString("\n")

	// Results list
	filtered := m.filteredResults()

	if len(filtered) == 0 && !m.isRunning {
		table.WriteString(styles.SubtleText.Render("  No submissions found"))
		table.WriteString("\n")
	}

	endIdx := m.scrollOffset + m.visibleRows
	if endIdx > len(filtered) {
		endIdx = len(filtered)
	}

	for i := m.scrollOffset; i < endIdx; i++ {
		r := filtered[i]

		// Cursor indicator
		var cursor string
		if i == m.cursor {
			cursor = styles.Highlight.Render("▶ ")
		} else {
			cursor = "  "
		}

		// Status icon
		var statusText string
		var statusStyled string
		switch r.Status {
		case domain.StatusClean:
			if r.Submission.HasMissingFiles() {
				statusText = "[~]"
				statusStyled = styles.WarningText.Render(statusText)
			} else {
				statusText = "[OK]"
				statusStyled = styles.SuccessText.Render(statusText)
			}
		case domain.StatusBanned:
			statusText = "[!]"
			statusStyled = styles.WarningText.Render(statusText)
		case domain.StatusFailed, domain.StatusTimedOut:
			statusText = "[X]"
			statusStyled = styles.ErrorText.Render(statusText)
		default:
			statusText = "..."
			statusStyled = statusText
		}
		// Pad status to fixed width
		statusPadding := strings.Repeat(" ", max(0, colStatus-lipgloss.Width(statusText)))

		// Truncate ID if needed 
		id := r.Submission.ID
		if m.settings.ShortNames {
			if idx := strings.Index(id, "_"); idx > 0 {
				id = id[:idx]
			}
		}
		// Truncate based on display width, not byte length
		if lipgloss.Width(id) > colSubmission {
			// Truncate rune by rune until it fits
			runes := []rune(id)
			for lipgloss.Width(string(runes)) > colSubmission-3 && len(runes) > 0 {
				runes = runes[:len(runes)-1]
			}
			id = string(runes) + "..."
		}
		// Pad ID to fixed width
		idPadding := strings.Repeat(" ", max(0, colSubmission-lipgloss.Width(id)))

		// Compile status
		var compileText, compileStyled string
		if r.Compile.TimedOut {
			compileText = "TIMEOUT"
			compileStyled = styles.WarningText.Render(compileText)
		} else if !r.Compile.OK {
			compileText = "FAIL"
			compileStyled = styles.ErrorText.Render(compileText)
		} else {
			compileText = "OK"
			compileStyled = styles.SuccessText.Render(compileText)
		}
		// Pad compile to fixed width
		compilePadding := strings.Repeat(" ", colCompile-len(compileText))

		// Banned count
		var bannedText, bannedStyled string
		if r.Scan.TotalHits() > 0 {
			bannedText = fmt.Sprintf("%d", r.Scan.TotalHits())
			bannedStyled = styles.WarningText.Render(bannedText)
		} else if r.Submission.HasMissingFiles() {
			bannedText = fmt.Sprintf("miss:%d", len(r.Submission.MissingFiles))
			bannedStyled = styles.WarningText.Render(bannedText)
		} else {
			bannedText = "-"
			bannedStyled = bannedText
		}
		// Pad banned to fixed width
		bannedPadding := strings.Repeat(" ", colBanned-len(bannedText))

		// Provisional Grade
		var gradeText, gradeStyled string
		if !r.Compile.OK || r.Compile.TimedOut || r.Scan.TotalHits() > 0 {
			gradeText = "2"
			gradeStyled = styles.ErrorText.Render(gradeText)
		} else {
			gradeText = "CHECK"
			gradeStyled = styles.SuccessText.Render(gradeText)
		}

		// Build row: cursor + status + padding + id + padding + compile + padding + banned + padding + grade
		table.WriteString(fmt.Sprintf("%s%s%s %s%s  %s%s  %s%s  %s\n",
			cursor,
			statusStyled, statusPadding,
			id, idPadding,
			compileStyled, compilePadding,
			bannedStyled, bannedPadding,
			gradeStyled))
	}

	// Scroll indicator
	if len(filtered) > m.visibleRows {
		table.WriteString(styles.SubtleText.Render(fmt.Sprintf("\n  Showing %d-%d of %d",
			m.scrollOffset+1, endIdx, len(filtered))))
	}

	b.WriteString(tableBox.Render(table.String()))

	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "details"},
		{Key: "f", Desc: "filter"},
		{Key: "r", Desc: "re-run"},
		{Key: "e", Desc: "export"},
		{Key: "esc", Desc: "back"},
	}))

	return b.String()
}

func (m Model) renderSummary() string {
	if m.report == nil {
		return ""
	}

	s := m.report.Summary
	var parts []string

	parts = append(parts, fmt.Sprintf("Total: %d", s.TotalSubmissions))
	parts = append(parts, styles.SuccessText.Render(fmt.Sprintf("Clean: %d", s.CleanSubmissions)))
	parts = append(parts, styles.WarningText.Render(fmt.Sprintf("Banned: %d", s.SubmissionsWithBanned)))
	parts = append(parts, styles.ErrorText.Render(fmt.Sprintf("Failed: %d", s.CompileFail)))

	return "  " + strings.Join(parts, "  •  ")
}

func (m Model) renderDetails() string {
	var b strings.Builder

	filtered := m.filteredResults()
	if m.cursor >= len(filtered) {
		return "No submission selected"
	}

	r := filtered[m.cursor]

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		Padding(1, 2)
	b.WriteString(header.Render(r.Submission.ID))
	b.WriteString("\n")

	// Tabs
	tabs := []string{"Compile", "Banned Calls", "Files"}
	var tabRow strings.Builder
	tabRow.WriteString("  ")
	for i, tab := range tabs {
		if i == m.detailsTab {
			tabRow.WriteString(styles.TabActive.Render(fmt.Sprintf(" %s ", tab)))
		} else {
			tabRow.WriteString(styles.TabInactive.Render(fmt.Sprintf(" %s ", tab)))
		}
		tabRow.WriteString(" ")
	}
	b.WriteString(tabRow.String())
	b.WriteString("\n\n")

	// Content box with border
	contentBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(1, 2).
		Width(min(m.width-4, 80))

	var content string
	switch m.detailsTab {
	case 0:
		content = m.renderCompileTab(r)
	case 1:
		content = m.renderBannedTab(r)
	case 2:
		content = m.renderFilesTab(r)
	}

	b.WriteString(contentBox.Render(content))

	b.WriteString("\n\n")

	// Show different help based on tab
	if m.detailsTab == 1 {
		b.WriteString(components.RenderHelpBar([]components.HelpItem{
			{Key: "tab", Desc: "switch tabs"},
			{Key: "↑/↓", Desc: "navigate"},
			{Key: "enter", Desc: "expand/collapse"},
			{Key: "esc", Desc: "back"},
		}))
	} else {
		b.WriteString(components.RenderHelpBar([]components.HelpItem{
			{Key: "tab", Desc: "switch tabs"},
			{Key: "↑/↓", Desc: "scroll"},
			{Key: "esc", Desc: "back"},
		}))
	}

	return b.String()
}

func (m Model) renderCompileTab(r domain.SubmissionResult) string {
	var b strings.Builder

	// Compile status - NO EMOJIS
	if r.Compile.OK {
		b.WriteString(styles.SuccessText.Render("[PASS] Compilation successful"))
	} else if r.Compile.TimedOut {
		b.WriteString(styles.ErrorText.Render("[TIMEOUT] Compilation timed out (5s limit)"))
	} else {
		b.WriteString(styles.ErrorText.Render(fmt.Sprintf("[FAIL] Compilation failed (exit %d)", r.Compile.ExitCode)))
	}
	b.WriteString("\n\n")

	// Command
	b.WriteString(styles.SubtleText.Render("Command:"))
	b.WriteString("\n")
	if len(r.Compile.Command) > 0 {
		cmd := strings.Join(r.Compile.Command, " ")
		if len(cmd) > 70 {
			cmd = cmd[:67] + "..."
		}
		b.WriteString(cmd)
		b.WriteString("\n")
	}

	// Output/Error
	if r.Compile.Stderr != "" {
		b.WriteString("\n")
		b.WriteString(styles.SubtleText.Render("Output:"))
		b.WriteString("\n")
		lines := strings.Split(r.Compile.Stderr, "\n")
		start := m.detailScroll
		end := start + 10
		if end > len(lines) {
			end = len(lines)
		}
		if start >= len(lines) {
			start = 0
		}
		for i := start; i < end; i++ {
			line := lines[i]
			if len(line) > 70 {
				line = line[:67] + "..."
			}
			b.WriteString(line + "\n")
		}
		if len(lines) > 10 {
			b.WriteString(styles.SubtleText.Render(fmt.Sprintf("\n(Showing %d-%d of %d lines)", start+1, end, len(lines))))
		}
	}

	return b.String()
}

func (m Model) renderBannedTab(r domain.SubmissionResult) string {
	var b strings.Builder

	if r.Scan.TotalHits() == 0 {
		b.WriteString(styles.SuccessText.Render("[OK] No banned function calls detected"))
		return b.String()
	}

	b.WriteString(styles.WarningText.Render(fmt.Sprintf("[!] %d banned call(s) found", r.Scan.TotalHits())))
	b.WriteString("\n\n")

	// Get sorted function names
	var funcNames []string
	for fn := range r.Scan.HitsByFunction {
		funcNames = append(funcNames, fn)
	}
	sort.Strings(funcNames)

	// Clamp cursor
	if m.bannedCursor >= len(funcNames) && len(funcNames) > 0 {
		// m.bannedCursor = len(funcNames) - 1
	}

	// Render functions with expandable details
	for i, fn := range funcNames {
		hits := r.Scan.HitsByFunction[fn]
		expanded := m.expandedFuncs != nil && m.expandedFuncs[fn]

		// Expand/collapse arrow
		arrow := "[+]"
		if expanded {
			arrow = "[-]"
		}

		// Build the complete line for this function header
		var line string
		if i == m.bannedCursor {
			line = "> " + styles.Highlight.Render(fmt.Sprintf("%s %s (%d)", arrow, fn, len(hits)))
		} else {
			line = fmt.Sprintf("  %s %s (%d)", arrow, fn, len(hits))
		}
		b.WriteString(line)
		b.WriteString("\n")

		// Show hits if expanded (max 5)
		if expanded {
			showMax := 5
			maxLineWidth := 65
			for j, hit := range hits {
				if j >= showMax {
					remaining := len(hits) - showMax
					b.WriteString(styles.SubtleText.Render(fmt.Sprintf("       ... and %d more calls", remaining)))
					b.WriteString("\n")
					break
				}
				// Build the hit line and truncate if too long (runes)
				hitLine := fmt.Sprintf("       %s:%d %s", hit.File, hit.Line, hit.Snippet)
				if lipgloss.Width(hitLine) > maxLineWidth {
					runes := []rune(hitLine)
					for lipgloss.Width(string(runes)) > maxLineWidth-3 && len(runes) > 0 {
						runes = runes[:len(runes)-1]
					}
					hitLine = string(runes) + "..."
				}
				b.WriteString(styles.SubtleText.Render(hitLine))
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

func (m Model) renderFilesTab(r domain.SubmissionResult) string {
	var b strings.Builder

	b.WriteString(styles.SubtleText.Render(fmt.Sprintf("%d source file(s)", len(r.Submission.CFiles))))
	b.WriteString("\n\n")

	for _, f := range r.Submission.CFiles {
		b.WriteString(fmt.Sprintf("  %s\n", f))
	}

	if len(r.Scan.ParseErrors) > 0 {
		b.WriteString("\n")
		b.WriteString(styles.WarningText.Render("Parse errors:"))
		b.WriteString("\n")
		for _, e := range r.Scan.ParseErrors {
			b.WriteString(fmt.Sprintf("  - %s\n", e))
		}
	}

	return b.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// Export View
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) renderExport() string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("Export Results"))
	b.WriteString("\n\n")

	boxWidth := min(50, m.width-4)

	// Format definitions
	formats := []struct {
		name string
		ext  string
		desc string
	}{
		{
			name: "JSON",
			ext:  ".json",
			desc: "Structured data for scripts & tools",
		},
		{
			name: "CSV",
			ext:  ".csv",
			desc: "Import into Excel, Google Sheets",
		},
	}

	for i, f := range formats {
		borderColor := styles.Muted
		if i == m.exportCursor {
			borderColor = styles.Primary
		}

		formatBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1, 2).
			Width(boxWidth)

		var content strings.Builder

		if i == m.exportCursor {
			content.WriteString("▸ ")
			content.WriteString(styles.SelectedItem.Render(f.name))
		} else {
			content.WriteString("  ")
			content.WriteString(styles.NormalItem.Render(f.name))
		}
		content.WriteString(styles.SubtleText.Render(fmt.Sprintf("  (%s)", f.ext)))
		content.WriteString("\n")

		content.WriteString(styles.SubtleText.Render("  " + f.desc))

		b.WriteString(formatBox.Render(content.String()))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.SubtleText.Render("  Output: ./autoscan_report" + formats[m.exportCursor].ext))
	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{"↑/↓", "navigate"},
		{"enter", "export"},
		{"esc", "back"},
	}))

	return b.String()
}

func (m Model) doExport() tea.Cmd {
	return func() tea.Msg {
		outputDir := "."
		var path string
		var err error
		var format string

		switch m.exportCursor {
		case 0:
			format = "JSON"
			path, err = export.JSON(*m.report, outputDir)
		case 1:
			format = "CSV"
			path, err = export.CSV(*m.report, outputDir)
		}

		if err != nil {
			return errorMsg(err)
		}
		return exportDoneMsg{format: format, path: path}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Banned Editor
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) renderBannedEditor() string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("Edit Banned Functions"))
	b.WriteString("\n\n")

	box := styles.BoxStyle(min(50, m.width-4))

	var content strings.Builder
	content.WriteString(styles.SubtleText.Render("Global banned function list"))
	content.WriteString("\n\n")

	if len(m.bannedList) == 0 {
		content.WriteString(styles.SubtleText.Render("  (no banned functions)"))
		content.WriteString("\n")
	} else {
		for i, fn := range m.bannedList {
			cursor := "  "
			style := styles.NormalItem
			if i == m.bannedCursorEdit {
				cursor = "▸ "
				style = styles.SelectedItem
			}

			if m.bannedEditing && i == m.bannedCursorEdit {
				content.WriteString(fmt.Sprintf("%s%s\n", cursor, m.bannedInput.View()))
			} else {
				content.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(fn)))
			}
		}
	}

	b.WriteString(box.Render(content.String()))

	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{"↑/↓", "navigate"},
		{"a", "add"},
		{"e/enter", "edit"},
		{"d", "delete"},
		{"esc", "save & exit"},
	}))

	return b.String()
}
