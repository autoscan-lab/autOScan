package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/feli05/autoscan/internal/domain"
	"github.com/feli05/autoscan/internal/export"
	"github.com/feli05/autoscan/internal/tui/components"
	"github.com/feli05/autoscan/internal/tui/styles"
	"github.com/feli05/autoscan/internal/tui/views/home"
	"github.com/feli05/autoscan/internal/tui/views/settings"
)

func (m Model) View() string {
	var content string

	switch m.currentView {
	case ViewHome:
		contentWidth := m.width - 4
		if contentWidth < 80 {
			contentWidth = 80
		}
		menuWidth := contentWidth * 55 / 100
		if menuWidth < 45 {
			menuWidth = 45
		}
		helpPanelWidth := contentWidth - menuWidth - 4
		if helpPanelWidth < 30 {
			helpPanelWidth = 30
		}
		m.helpPanel.SetWidth(helpPanelWidth)
		m.helpPanel.SetPolicyCount(len(m.policies))

		content = home.View(home.State{
			Width:         m.width,
			MenuItem:      int(m.menuItem),
			ConfirmDelete: m.confirmDelete,
			PolicyCount:   len(m.policies),
			AnimationView: m.eyeAnimation.View(),
			HelpPanelView: m.helpPanel.View(),
		})
	case ViewPolicySelect:
		content = m.renderPolicySelect()
	case ViewPolicyManage:
		content = m.renderPolicyManage()
	case ViewPolicyEditor:
		// Only add help bar if NOT in a sub-mode (sub-modes render their own hints)
		if m.policyEditor.InSubMode() {
			content = m.policyEditor.View()
		} else {
			content = m.policyEditor.View() + "\n\n" + components.RenderHelpBar([]components.HelpItem{
				{Key: "tab", Desc: "next field"},
				{Key: "↑↓", Desc: "navigate"},
				{Key: "esc", Desc: "cancel"},
			})
		}
	case ViewBannedEditor:
		content = m.renderBannedEditor()
	case ViewSettings:
		content = settings.View(settings.State{
			Settings:       &m.settings,
			SettingsCursor: m.settingsCursor,
			Width:          m.width,
		})
	case ViewDirectoryInput:
		content = m.renderDirectoryInput()
	case ViewSubmissions:
		content = m.renderSubmissions()
	case ViewDetails:
		content = m.renderDetails()
	case ViewExport:
		content = m.renderExport()
	default:
		contentWidth := m.width - 4
		if contentWidth < 80 {
			contentWidth = 80
		}
		menuWidth := contentWidth * 55 / 100
		if menuWidth < 45 {
			menuWidth = 45
		}
		helpPanelWidth := contentWidth - menuWidth - 4
		if helpPanelWidth < 30 {
			helpPanelWidth = 30
		}
		m.helpPanel.SetWidth(helpPanelWidth)
		m.helpPanel.SetPolicyCount(len(m.policies))

		content = home.View(home.State{
			Width:         m.width,
			MenuItem:      int(m.menuItem),
			ConfirmDelete: m.confirmDelete,
			PolicyCount:   len(m.policies),
			AnimationView: m.eyeAnimation.View(),
			HelpPanelView: m.helpPanel.View(),
		})
	}

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Left,
		lipgloss.Top,
		content,
	)
}

func (m Model) renderPolicySelect() string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("Select a Policy"))
	b.WriteString("\n\n")

	boxWidth := components.BoxWidth(m.width, 8, 60)

	if len(m.policies) == 0 {
		box := styles.WarningBoxStyle(boxWidth)
		content := styles.WarningText.Render("No policies found!") + "\n\n" +
			styles.SubtleText.Render("Create a policy via Manage Policies or edit ~/.config/autoscan/")
		b.WriteString(box.Render(content))
	} else {
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Primary).
			Padding(1, 2).
			Width(boxWidth)

		var list strings.Builder

		list.WriteString(styles.SubtleText.Render(fmt.Sprintf("Available policies: %d", len(m.policies))))
		list.WriteString("\n\n")

		for i, p := range m.policies {
			list.WriteString(components.RenderMenuItem(p.Name, i == m.selectedPolicy))
			list.WriteString("\n")
		}

		b.WriteString(box.Render(list.String()))

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

			details.WriteString(styles.SubtleText.Render("  Name:     "))
			details.WriteString(p.Name)
			details.WriteString("\n")

			relPath, _ := filepath.Rel(".", p.FilePath)
			details.WriteString(styles.SubtleText.Render("  File:     "))
			details.WriteString(filepath.Base(relPath))
			details.WriteString("\n")

			isMultiProcess := p.Run.MultiProcess != nil && p.Run.MultiProcess.Enabled
			details.WriteString(styles.SubtleText.Render("  Mode:     "))
			if isMultiProcess {
				details.WriteString(styles.SuccessText.Render("Multi-Process"))
			} else {
				details.WriteString("Single Process")
			}
			details.WriteString("\n")

			details.WriteString(styles.SubtleText.Render("  Flags:    "))
			if len(p.Compile.Flags) > 0 {
				details.WriteString(strings.Join(p.Compile.Flags, " "))
			} else {
				details.WriteString(styles.SubtleText.Render("(default)"))
			}
			details.WriteString("\n")

			if len(p.LibraryFiles) > 0 {
				details.WriteString(styles.SubtleText.Render("  Libraries:"))
				details.WriteString(strings.Join(p.LibraryFiles, ", "))
				details.WriteString("\n")
			}

			details.WriteString("\n")

			if isMultiProcess {
				mp := p.Run.MultiProcess
				details.WriteString(styles.PrimaryText.Render("  Executables"))
				details.WriteString("\n")
				for _, proc := range mp.Executables {
					details.WriteString(fmt.Sprintf("    • %s ", proc.Name))
					details.WriteString(styles.SubtleText.Render(fmt.Sprintf("(%s)", proc.SourceFile)))
					if proc.StartDelayMs > 0 {
						details.WriteString(styles.SubtleText.Render(fmt.Sprintf(" +%dms", proc.StartDelayMs)))
					}
					details.WriteString("\n")
				}

				if len(mp.TestScenarios) > 0 {
					details.WriteString("\n")
					details.WriteString(styles.PrimaryText.Render(fmt.Sprintf("  Test Scenarios (%d)", len(mp.TestScenarios))))
					details.WriteString("\n")
					for i, scenario := range mp.TestScenarios {
						if i >= 3 {
							details.WriteString(styles.SubtleText.Render(fmt.Sprintf("    ... and %d more", len(mp.TestScenarios)-3)))
							details.WriteString("\n")
							break
						}
						details.WriteString(fmt.Sprintf("    • %s\n", scenario.Name))
					}
				}
			} else {
				if p.Compile.SourceFile != "" {
					details.WriteString(styles.SubtleText.Render("  Source:   "))
					details.WriteString(p.Compile.SourceFile)
					details.WriteString("\n")
				}

				if len(p.Run.TestCases) > 0 {
					details.WriteString(styles.PrimaryText.Render(fmt.Sprintf("  Test Cases (%d)", len(p.Run.TestCases))))
					details.WriteString("\n")
					for i, tc := range p.Run.TestCases {
						if i >= 3 {
							details.WriteString(styles.SubtleText.Render(fmt.Sprintf("    ... and %d more", len(p.Run.TestCases)-3)))
							details.WriteString("\n")
							break
						}
						details.WriteString(fmt.Sprintf("    • %s\n", tc.Name))
					}
				} else {
					details.WriteString(styles.SubtleText.Render("  No test cases defined"))
					details.WriteString("\n")
				}
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

	if m.inputError != "" {
		b.WriteString("\n")
		b.WriteString(styles.ErrorText.Render("  " + m.inputError))
	}

	return b.String()
}

func (m Model) renderPolicyManage() string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("Manage Policies"))
	b.WriteString("\n\n")

	boxWidth := components.BoxWidth(m.width, 8, 60)

	configBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(1, 2).
		Width(boxWidth)

	var configSection strings.Builder
	configSection.WriteString(styles.SubtleText.Render("Configuration"))
	configSection.WriteString("\n\n")

	configSection.WriteString(components.RenderMenuItem("Edit Banned Functions", m.policyManageCursor == -1))
	configSection.WriteString("\n")
	configSection.WriteString(styles.SubtleText.Render("    Global list of prohibited function calls"))

	b.WriteString(configBox.Render(configSection.String()))
	b.WriteString("\n\n")

	policyBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2).
		Width(boxWidth)

	var policySection strings.Builder
	policySection.WriteString(styles.PrimaryText.Render(fmt.Sprintf("Policies (%d)", len(m.policies))))
	policySection.WriteString("\n\n")

	policySection.WriteString(components.RenderMenuItem("+ Create New Policy", m.policyManageCursor == 0))
	policySection.WriteString("\n")

	if len(m.policies) > 0 {
		policySection.WriteString("\n")
	}

	for i, p := range m.policies {
		policySection.WriteString(components.RenderMenuItem(p.Name, m.policyManageCursor == i+1))
		policySection.WriteString("\n")
	}

	if m.policyManageCursor > 0 && m.policyManageCursor <= len(m.policies) {
		p := m.policies[m.policyManageCursor-1]
		policySection.WriteString("\n")
		policySection.WriteString(styles.SubtleText.Render(fmt.Sprintf("  File: %s", filepath.Base(p.FilePath))))
		if len(p.Compile.Flags) > 0 {
			policySection.WriteString("\n")
			policySection.WriteString(styles.SubtleText.Render(fmt.Sprintf("  Flags: %s", strings.Join(p.Compile.Flags, " "))))
		}
	}

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

func (m Model) renderDirectoryInput() string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("Select Directory"))
	b.WriteString("\n\n")

	boxWidth := components.BoxWidth(m.width, 8, 60)
	box := styles.BoxStyle(boxWidth)

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

func (m Model) renderSubmissions() string {
	var b strings.Builder

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
	b.WriteString("\n\n")

	tabs := []string{"Results", "Similarity"}
	var tabRow strings.Builder
	tabRow.WriteString("  ")
	for i, tab := range tabs {
		if i == m.submissionsTab {
			tabRow.WriteString(styles.TabActive.Render(fmt.Sprintf(" %s ", tab)))
		} else {
			tabRow.WriteString(styles.TabInactive.Render(fmt.Sprintf(" %s ", tab)))
		}
		tabRow.WriteString(" ")
	}
	b.WriteString(tabRow.String())
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
		b.WriteString(m.renderSubmissionsHeaderBox())
		b.WriteString("\n")
	}

	b.WriteString("\n")

	if m.pairDetailOpen {
		b.WriteString(m.renderPairDetail())
		return b.String()
	}
	if m.submissionsTab == 0 {
		b.WriteString(m.renderSubmissionsResults())
	} else {
		b.WriteString(m.renderSubmissionsSimilarity())
	}

	return b.String()
}

func (m Model) renderSubmissionsHeaderBox() string {
	statsBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(0, 2).
		MarginTop(1)

	if m.submissionsTab == 0 {
		searchLabel := ""
		if strings.TrimSpace(m.searchQuery) != "" {
			searchLabel = fmt.Sprintf("  Search: %s", m.searchQuery)
		}

		stats := fmt.Sprintf(
			"Pass: %d  Fail: %d  Banned: %d  Time: %dms  Filter: %s%s",
			m.report.Summary.CompilePass,
			m.report.Summary.CompileFail,
			m.report.Summary.SubmissionsWithBanned,
			m.report.Summary.DurationMs,
			m.filter.String(),
			searchLabel,
		)
		return statsBox.Render(stats)
	}

	if len(m.similarityProcessNames) == 0 {
		return statsBox.Render("Similarity: no processes configured")
	}

	line2 := fmt.Sprintf(
		"Window size: %d   Min tokens: %d   Threshold: %.2f",
		m.settings.PlagiarismWindowSize,
		m.settings.PlagiarismMinFuncTokens,
		m.settings.PlagiarismScoreThreshold,
	)

	return statsBox.Render(styles.NormalItem.Render(line2))
}

func (m Model) renderSubmissionsResults() string {
	var b strings.Builder

	searchBorderColor := styles.Muted
	if m.searchActive {
		searchBorderColor = styles.Primary
	}
	searchBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(searchBorderColor).
		Padding(0, 2)
	b.WriteString(searchBox.Render(fmt.Sprintf("Search: %s", m.searchInput.View())))
	b.WriteString("\n\n")

	tableBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(0, 1)

	var table strings.Builder

	const (
		colStatus  = 5 // [OK], [!], [X], [~] + space
		colCompile = 10
		colBanned  = 10
		colGrade   = 8
	)
	fixedCols := colStatus + colCompile + colBanned + colGrade + 15
	colSubmission := m.width - fixedCols
	if colSubmission < 30 {
		colSubmission = 30
	}
	if colSubmission > 80 {
		colSubmission = 80
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary)

	table.WriteString(headerStyle.Render(fmt.Sprintf("  %-*s %-*s  %-*s  %-*s  %-*s",
		colStatus, "",
		colSubmission, "Submission",
		colCompile, "Compile",
		colBanned, "Banned",
		colGrade, "Grade")))
	table.WriteString("\n")
	table.WriteString(strings.Repeat("─", 2+colStatus+1+colSubmission+2+colCompile+2+colBanned+2+colGrade))
	table.WriteString("\n")

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

		var cursor string
		if i == m.cursor {
			cursor = styles.Highlight.Render("▶ ")
		} else {
			cursor = "  "
		}

		var statusText string
		var statusStyled string
		switch r.Status {
		case domain.StatusClean:
			statusText = "[OK]"
			statusStyled = styles.SuccessText.Render(statusText)
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
		statusPadding := strings.Repeat(" ", max(0, colStatus-lipgloss.Width(statusText)))

		id := r.Submission.ID
		if m.settings.ShortNames {
			if idx := strings.Index(id, "_"); idx > 0 {
				id = id[:idx]
			}
		}
		if lipgloss.Width(id) > colSubmission {
			runes := []rune(id)
			for lipgloss.Width(string(runes)) > colSubmission-3 && len(runes) > 0 {
				runes = runes[:len(runes)-1]
			}
			id = string(runes) + "..."
		}
		idPadding := strings.Repeat(" ", max(0, colSubmission-lipgloss.Width(id)))

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
		compilePadding := strings.Repeat(" ", colCompile-len(compileText))

		var bannedText, bannedStyled string
		if r.Scan.TotalHits() > 0 {
			bannedText = fmt.Sprintf("%d", r.Scan.TotalHits())
			bannedStyled = styles.WarningText.Render(bannedText)
		} else {
			bannedText = "-"
			bannedStyled = bannedText
		}
		bannedPadding := strings.Repeat(" ", colBanned-len(bannedText))

		var gradeText, gradeStyled string
		if !r.Compile.OK || r.Compile.TimedOut || r.Scan.TotalHits() > 0 {
			gradeText = "2"
			gradeStyled = styles.ErrorText.Render(gradeText)
		} else {
			gradeText = "CHECK"
			gradeStyled = styles.SuccessText.Render(gradeText)
		}

		table.WriteString(fmt.Sprintf("%s%s%s %s%s  %s%s  %s%s  %s\n",
			cursor,
			statusStyled, statusPadding,
			id, idPadding,
			compileStyled, compilePadding,
			bannedStyled, bannedPadding,
			gradeStyled))
	}

	if len(filtered) > m.visibleRows {
		table.WriteString(styles.SubtleText.Render(fmt.Sprintf("\n  Showing %d-%d of %d",
			m.scrollOffset+1, endIdx, len(filtered))))
	}

	b.WriteString(tableBox.Render(table.String()))

	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{Key: "tab", Desc: "switch to similarity"},
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "details"},
		{Key: "/", Desc: "search"},
		{Key: "f", Desc: "filter"},
		{Key: "r", Desc: "re-run"},
		{Key: "e", Desc: "export"},
		{Key: "esc", Desc: "clear/back"},
	}))

	return b.String()
}

func (m Model) renderSubmissionsSimilarity() string {
	var b strings.Builder

	if m.report == nil {
		b.WriteString(styles.SubtleText.Render("No run data available. Run the grader first."))
		return b.String()
	}

	if len(m.similarityProcessNames) == 0 {
		b.WriteString(styles.SubtleText.Render("No processes configured. Check policy configuration."))
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString("  ")
	b.WriteString(styles.SubtleText.Render("Process: "))
	for i, name := range m.similarityProcessNames {
		if i == m.similaritySelectedProc {
			b.WriteString(styles.TabActive.Render(fmt.Sprintf(" %s ", name)))
		} else {
			b.WriteString(styles.TabInactive.Render(fmt.Sprintf(" %s ", name)))
		}
		b.WriteString(" ")
	}
	b.WriteString("\n\n")

	currentProc := m.currentSimilarityProcessName()
	pairs := m.similarityPairsByProcess[currentProc]
	state := m.similarityStateByProcess[currentProc]

	if errText, ok := m.similarityErrorByProcess[currentProc]; ok && errText != "" {
		b.WriteString(styles.WarningText.Render("Similarity error: " + errText))
		b.WriteString("\n")
		return b.String()
	}

	if state == SimilarityNotStarted || state == SimilarityComputing {
		b.WriteString(styles.SubtleText.Render("Computing similarity..."))
		b.WriteString("\n")
		return b.String()
	}
	if len(pairs) == 0 {
		b.WriteString(styles.SubtleText.Render("No pairs found (not enough comparable submissions)."))
		b.WriteString("\n")
		return b.String()
	}

	tableBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(0, 1)

	var table strings.Builder

	padOrTrim := func(s string, w int) string {
		s = components.TruncateToWidth(s, w)
		if d := w - lipgloss.Width(s); d > 0 {
			s += strings.Repeat(" ", d)
		}
		return s
	}

	const (
		colRank    = 5
		colJac     = 9
		colPerFunc = 9
		colMatches = 13
		colStatus  = 8
	)
	fixedCols := 2 + colRank + colJac + colPerFunc + colMatches + colStatus + 7
	availForNames := m.width - fixedCols
	colSub := 20
	colSubB := 20
	if availForNames > 0 {
		per := availForNames / 2
		if per > colSub {
			colSub = per
			colSubB = per
		}
	}
	if colSub > 34 {
		colSub = 34
	}
	if colSubB > 34 {
		colSubB = 34
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary)

	headerLine := "  " +
		padOrTrim("#", colRank) + " " +
		padOrTrim("Submission A", colSub) + " " +
		padOrTrim("Submission B", colSubB) + " " +
		padOrTrim("Jaccard", colJac) + " " +
		padOrTrim("Per-func", colPerFunc) + " " +
		padOrTrim("Matches", colMatches) + " " +
		padOrTrim("Status", colStatus)
	table.WriteString(headerStyle.Render(headerLine))
	table.WriteString("\n")
	table.WriteString(strings.Repeat("─", 2+colRank+1+colSub+1+colSubB+1+colJac+1+colPerFunc+1+colMatches+1+colStatus))
	table.WriteString("\n")

	dataRows := min(30, m.visibleRows-1) // reserve 1 line for footer/padding
	if dataRows < 6 {
		dataRows = 6
	}

	endIdx := m.similarityScroll + dataRows
	if endIdx > len(pairs) {
		endIdx = len(pairs)
	}

	for i := m.similarityScroll; i < endIdx; i++ {
		p := pairs[i]
		res := p.Result

		cursor := "  "
		if i == m.similarityCursor {
			cursor = styles.Highlight.Render("▶ ")
		}

		rank := i + 1
		aID := m.results[p.AIndex].Submission.ID
		bID := m.results[p.BIndex].Submission.ID
		if m.settings.ShortNames {
			if idx := strings.Index(aID, "_"); idx > 0 {
				aID = aID[:idx]
			}
			if idx := strings.Index(bID, "_"); idx > 0 {
				bID = bID[:idx]
			}
		}
		if lipgloss.Width(aID) > colSub {
			runes := []rune(aID)
			for lipgloss.Width(string(runes)) > colSub-3 && len(runes) > 0 {
				runes = runes[:len(runes)-1]
			}
			aID = string(runes) + "..."
		}
		if lipgloss.Width(bID) > colSubB {
			runes := []rune(bID)
			for lipgloss.Width(string(runes)) > colSubB-3 && len(runes) > 0 {
				runes = runes[:len(runes)-1]
			}
			bID = string(runes) + "..."
		}

		statusText := "OK"
		if res.Flagged {
			statusText = "FLAG"
		}

		matchesText := fmt.Sprintf("%d/%d", res.WindowMatches, res.WindowUnion)
		jacText := fmt.Sprintf("%.2f%%", res.WindowJaccard*100)
		perFuncText := fmt.Sprintf("%.2f%%", res.PerFuncSimilarity*100)

		statusPadded := padOrTrim(statusText, colStatus)
		statusRendered := styles.SuccessText.Render(statusPadded)
		if res.Flagged {
			statusRendered = styles.WarningText.Render(statusPadded)
		}

		row := cursor +
			padOrTrim(fmt.Sprintf("%d", rank), colRank) + " " +
			padOrTrim(aID, colSub) + " " +
			padOrTrim(bID, colSubB) + " " +
			padOrTrim(jacText, colJac) + " " +
			padOrTrim(perFuncText, colPerFunc) + " " +
			padOrTrim(matchesText, colMatches) + " " +
			statusRendered
		table.WriteString(row)
		table.WriteString("\n")
	}

	for i := endIdx; i < m.similarityScroll+dataRows; i++ {
		table.WriteString("\n")
	}

	footer := ""
	if len(pairs) > dataRows {
		footer = fmt.Sprintf("  Showing %d-%d of %d", m.similarityScroll+1, endIdx, len(pairs))
	}
	table.WriteString(styles.SubtleText.Render(padOrTrim(footer, 2+colRank+1+colSub+1+colSubB+1+colJac+1+colPerFunc+1+colMatches+1+colStatus)))

	b.WriteString(tableBox.Render(table.String()))
	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{Key: "tab", Desc: "switch to results"},
		{Key: "h/l", Desc: "prev/next process"},
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "pair detail"},
		{Key: "esc", Desc: "back"},
	}))

	return b.String()
}

var pairDetailHighlightStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#334155")).
	Foreground(styles.Text)

func (m Model) renderPairDetail() string {
	var b strings.Builder

	pairs := m.similarityPairsByProcess[m.pairDetailProcess]
	if m.pairDetailPairIndex >= len(pairs) {
		b.WriteString(styles.SubtleText.Render("No pair selected."))
		return b.String()
	}
	pair := pairs[m.pairDetailPairIndex]
	res := pair.Result

	if m.pairDetailLoadErr != "" {
		b.WriteString(styles.WarningText.Render("Error: " + m.pairDetailLoadErr))
		b.WriteString("\n\n")
		b.WriteString(components.RenderHelpBar([]components.HelpItem{{Key: "esc", Desc: "back"}}))
		return b.String()
	}
	if m.pairDetailContentA == nil {
		b.WriteString(styles.SubtleText.Render("Loading files..."))
		b.WriteString("\n\n")
		b.WriteString(components.RenderHelpBar([]components.HelpItem{{Key: "esc", Desc: "back"}}))
		return b.String()
	}

	statsBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(0, 2).
		MarginTop(0)
	nameA := m.results[pair.AIndex].Submission.ID
	nameB := m.results[pair.BIndex].Submission.ID
	if m.settings.ShortNames {
		if idx := strings.Index(nameA, "_"); idx > 0 {
			nameA = nameA[:idx]
		}
		if idx := strings.Index(nameB, "_"); idx > 0 {
			nameB = nameB[:idx]
		}
	}
	summary := fmt.Sprintf(
		"%s  vs  %s   ·   Jaccard: %.2f%%   Per-func: %.2f%%   Matches: %d/%d",
		nameA, nameB,
		res.WindowJaccard*100, res.PerFuncSimilarity*100,
		res.WindowMatches, res.WindowUnion,
	)
	b.WriteString(statsBox.Render(summary))
	b.WriteString("\n\n")

	const pairDetailMaxPaneHeight = 30
	paneHeight := min(m.visibleRows, pairDetailMaxPaneHeight)
	if paneHeight < 8 {
		paneHeight = 8
	}

	var spansA, spansB []domain.MatchSpan
	for _, wm := range res.Matches {
		spansA = append(spansA, wm.SpansA...)
		spansB = append(spansB, wm.SpansB...)
	}

	halfWidth, contentWidth := pairDetailPaneWidths(m.width)

	leftPane := renderCodePane(m.pairDetailContentA, spansA, m.pairDetailScrollA, m.pairDetailHScrollA, paneHeight, contentWidth)
	rightPane := renderCodePane(m.pairDetailContentB, spansB, m.pairDetailScrollB, m.pairDetailHScrollB, paneHeight, contentWidth)

	// Same technique as renderDiffLines: constrain each line with Width/MaxWidth before putting in box
	lineStyle := lipgloss.NewStyle().Width(contentWidth).MaxWidth(contentWidth)
	leftLines := strings.Split(strings.TrimSuffix(leftPane, "\n"), "\n")
	rightLines := strings.Split(strings.TrimSuffix(rightPane, "\n"), "\n")
	for len(leftLines) < paneHeight {
		leftLines = append(leftLines, strings.Repeat(" ", contentWidth))
	}
	for len(rightLines) < paneHeight {
		rightLines = append(rightLines, strings.Repeat(" ", contentWidth))
	}

	var leftContent, rightContent strings.Builder
	for i := 0; i < paneHeight; i++ {
		leftContent.WriteString(lineStyle.Render(leftLines[i]))
		leftContent.WriteString("\n")
		rightContent.WriteString(lineStyle.Render(rightLines[i]))
		rightContent.WriteString("\n")
	}

	leftBorderColor := styles.Muted
	rightBorderColor := styles.Muted
	if m.pairDetailFocusedPane == 0 {
		leftBorderColor = styles.Primary
	} else {
		rightBorderColor = styles.Primary
	}

	leftBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(leftBorderColor).
		Padding(0, 1).
		Width(halfWidth).
		Render(strings.TrimSuffix(leftContent.String(), "\n"))
	rightBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(rightBorderColor).
		Padding(0, 1).
		Width(halfWidth).
		Render(strings.TrimSuffix(rightContent.String(), "\n"))

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftBox, "  ", rightBox))

	matchInfo := ""
	if len(res.Matches) > 0 {
		matchInfo = fmt.Sprintf("  %d matches", len(res.Matches))
	}
	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{Key: "↑/↓", Desc: "scroll"},
		{Key: "←/→", Desc: "pan"},
		{Key: "h/l", Desc: "switch pane"},
		{Key: "esc", Desc: "back"},
	}))
	if matchInfo != "" {
		b.WriteString(styles.SubtleText.Render(matchInfo))
	}
	return b.String()
}

const pairDetailTabWidth = 8

func pairDetailPaneWidths(totalWidth int) (halfWidth, contentWidth int) {
	halfWidth = (totalWidth - 6) / 2
	if halfWidth < 20 {
		halfWidth = 20
	}
	// Same as renderProcessBox/renderExecuteResult: content width = box width - border - padding
	contentWidth = halfWidth - 4
	if contentWidth < 10 {
		contentWidth = 10
	}
	return halfWidth, contentWidth
}

func expandTabsForPane(s string, width int) string {
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

func sliceDisplayWindow(s string, start, width int) string {
	if width <= 0 || s == "" {
		return ""
	}
	if start < 0 {
		start = 0
	}
	var b strings.Builder
	b.Grow(len(s))
	col := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if col+rw <= start {
			col += rw
			continue
		}
		if col-start+rw > width {
			break
		}
		b.WriteRune(r)
		col += rw
	}
	return b.String()
}

func byteColToDisplayCol(line string, byteCol, tabWidth int) int {
	if byteCol <= 1 {
		return 1
	}
	target := byteCol - 1
	col := 1
	for idx, r := range line {
		if idx >= target {
			break
		}
		if r == '\t' {
			spaces := tabWidth - ((col - 1) % tabWidth)
			if spaces == 0 {
				spaces = tabWidth
			}
			col += spaces
			continue
		}
		col += lipgloss.Width(string(r))
	}
	return col
}

func renderCodePane(content []byte, spans []domain.MatchSpan, scrollLine, hScroll, height, width int) string {
	lines := strings.Split(string(content), "\n")
	blankLine := strings.Repeat(" ", width) + "\n"

	var b strings.Builder
	contentLines := 0
	if scrollLine < len(lines) {
		end := scrollLine + height
		if end > len(lines) {
			end = len(lines)
		}
		contentLines = end - scrollLine
		for i := scrollLine; i < end; i++ {
			rawLine := components.SanitizeDisplay(lines[i])
			line := expandTabsForPane(rawLine, pairDetailTabWidth)
			fullLineWidth := lipgloss.Width(line)
			line = sliceDisplayWindow(line, hScroll, width)
			lineNum1 := i + 1
			runes := []rune(line)
			lineRuneLen := len(runes)
			var ranges [][2]int
			for _, sp := range spans {
				if sp.EndLine < lineNum1 || sp.StartLine > lineNum1 {
					continue
				}
				startCol := byteColToDisplayCol(rawLine, sp.StartCol, pairDetailTabWidth)
				endCol := byteColToDisplayCol(rawLine, sp.EndCol, pairDetailTabWidth)
				if sp.StartLine == lineNum1 && sp.EndLine == lineNum1 {
					// segment on this line
				} else if sp.StartLine == lineNum1 {
					endCol = fullLineWidth + 1
				} else if sp.EndLine == lineNum1 {
					startCol = 1
				} else {
					startCol = 1
					endCol = fullLineWidth + 1
				}
				startIdx := (startCol - 1) - hScroll
				if startIdx < 0 {
					startIdx = 0
				}
				endIdx := endCol - hScroll
				if endIdx > lineRuneLen {
					endIdx = lineRuneLen
				}
				if startIdx < endIdx {
					ranges = append(ranges, [2]int{startIdx, endIdx})
				}
			}
			merged := mergeRanges(ranges)
			if len(merged) == 0 {
				b.WriteString(string(runes))
			} else {
				last := 0
				for _, r := range merged {
					if r[0] > last {
						b.WriteString(string(runes[last:r[0]]))
					}
					seg := string(runes[r[0]:r[1]])
					b.WriteString(pairDetailHighlightStyle.Render(seg))
					last = r[1]
				}
				if last < lineRuneLen {
					b.WriteString(string(runes[last:]))
				}
			}
			lineWidth := lipgloss.Width(line)
			if lineWidth < width {
				b.WriteString(strings.Repeat(" ", width-lineWidth))
			}
			b.WriteString("\n")
		}
	}
	for i := contentLines; i < height; i++ {
		b.WriteString(blankLine)
	}
	return b.String()
}

func mergeRanges(ranges [][2]int) [][2]int {
	if len(ranges) == 0 {
		return nil
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i][0] < ranges[j][0] })
	out := [][2]int{ranges[0]}
	for i := 1; i < len(ranges); i++ {
		cur := &out[len(out)-1]
		if ranges[i][0] <= cur[1] {
			if ranges[i][1] > cur[1] {
				cur[1] = ranges[i][1]
			}
			continue
		}
		out = append(out, ranges[i])
	}
	return out
}

func (m Model) renderDetails() string {
	var b strings.Builder

	filtered := m.filteredResults()
	if m.cursor >= len(filtered) {
		return "No submission selected"
	}

	r := filtered[m.cursor]

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		Padding(1, 2)
	b.WriteString(header.Render(r.Submission.ID))
	b.WriteString("\n")

	tabs := []string{"Compile", "Banned", "Files", "Run"}
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

	contentWidth := m.width - 8
	if contentWidth < 80 {
		contentWidth = 80
	}
	contentBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(1, 2).
		Width(contentWidth)

	var content string
	switch m.detailsTab {
	case 0:
		content = m.renderCompileTab(r)
	case 1:
		content = m.renderBannedTab(r)
	case 2:
		content = m.renderFilesTab(r)
	case 3:
		content = m.renderRunTab(r)
	}

	b.WriteString(contentBox.Render(content))

	b.WriteString("\n\n")

	switch m.detailsTab {
	case 1:
		b.WriteString(components.RenderHelpBar([]components.HelpItem{
			{Key: "tab", Desc: "switch tabs"},
			{Key: "↑/↓", Desc: "navigate"},
			{Key: "enter", Desc: "expand/collapse"},
			{Key: "esc", Desc: "back"},
		}))
	case 3:
		helpItems := []components.HelpItem{
			{Key: "tab", Desc: "switch tabs"},
			{Key: "↑/↓", Desc: "navigate"},
			{Key: "enter", Desc: "run/focus"},
		}
		if m.selectedPolicy >= 0 && m.selectedPolicy < len(m.policies) {
			if mp := m.policies[m.selectedPolicy].Run.MultiProcess; mp != nil && mp.Enabled {
				helpItems = append(helpItems, components.HelpItem{Key: "m", Desc: "multi-process"})
			}
		}
		helpItems = append(helpItems, components.HelpItem{Key: "esc", Desc: "back"})
		b.WriteString(components.RenderHelpBar(helpItems))
	default:
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

	availableWidth := m.width - 12
	if availableWidth < 60 {
		availableWidth = 60
	}

	if r.Compile.OK {
		b.WriteString(styles.SuccessText.Render("[PASS] Compilation successful"))
	} else if r.Compile.TimedOut {
		b.WriteString(styles.ErrorText.Render("[TIMEOUT] Compilation timed out (5s limit)"))
	} else {
		b.WriteString(styles.ErrorText.Render(fmt.Sprintf("[FAIL] Compilation failed (exit %d)", r.Compile.ExitCode)))
	}
	b.WriteString("\n\n")

	b.WriteString(styles.SubtleText.Render("Command:"))
	b.WriteString("\n")
	if len(r.Compile.Command) > 0 {
		var truncatedCmd []string
		for _, arg := range r.Compile.Command {
			truncatedCmd = append(truncatedCmd, truncatePathToFilename(arg))
		}
		cmd := strings.Join(truncatedCmd, " ")
		cmdStyle := lipgloss.NewStyle().Width(availableWidth)
		b.WriteString(cmdStyle.Render(cmd))
		b.WriteString("\n")
	}

	if r.Compile.Stderr != "" {
		b.WriteString("\n")
		b.WriteString(styles.SubtleText.Render("Output:"))
		b.WriteString("\n")
		truncatedStderr := truncatePathsInText(r.Compile.Stderr)
		lines := strings.Split(truncatedStderr, "\n")
		start := m.detailScroll
		visibleLines := (m.height - 20)
		if visibleLines < 15 {
			visibleLines = 15
		}
		end := start + visibleLines
		if end > len(lines) {
			end = len(lines)
		}
		if start >= len(lines) {
			start = 0
		}

		lineStyle := lipgloss.NewStyle().Width(availableWidth)
		for i := start; i < end; i++ {
			line := lines[i]
			wrapped := lineStyle.Render(line)
			b.WriteString(wrapped)
			b.WriteString("\n")
		}
		if len(lines) > visibleLines {
			b.WriteString(styles.SubtleText.Render(fmt.Sprintf("\n(Showing %d-%d of %d lines, ↑/↓ to scroll)", start+1, end, len(lines))))
		}
	}

	return b.String()
}

func truncatePathToFilename(s string) string {
	if strings.Contains(s, "/") && !strings.HasPrefix(s, "-") {
		return filepath.Base(s)
	}
	return s
}

func truncatePathsInText(text string) string {
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

func (m Model) renderBannedTab(r domain.SubmissionResult) string {
	var b strings.Builder

	if r.Scan.TotalHits() == 0 {
		b.WriteString(styles.SuccessText.Render("[OK] No banned function calls detected"))
		return b.String()
	}

	b.WriteString(styles.WarningText.Render(fmt.Sprintf("[!] %d banned call(s) found", r.Scan.TotalHits())))
	b.WriteString("\n\n")

	var funcNames []string
	for fn := range r.Scan.HitsByFunction {
		funcNames = append(funcNames, fn)
	}
	sort.Strings(funcNames)

	for i, fn := range funcNames {
		hits := r.Scan.HitsByFunction[fn]
		expanded := m.expandedFuncs != nil && m.expandedFuncs[fn]

		arrow := "[+]"
		if expanded {
			arrow = "[-]"
		}

		var line string
		if i == m.bannedCursor {
			line = "> " + styles.Highlight.Render(fmt.Sprintf("%s %s (%d)", arrow, fn, len(hits)))
		} else {
			line = fmt.Sprintf("  %s %s (%d)", arrow, fn, len(hits))
		}
		b.WriteString(line)
		b.WriteString("\n")

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

func (m Model) renderRunTab(r domain.SubmissionResult) string {
	var b strings.Builder

	if !r.Compile.OK {
		b.WriteString(styles.ErrorText.Render("[!] Cannot run - compilation failed"))
		b.WriteString("\n\n")
		b.WriteString(styles.SubtleText.Render("Fix compilation errors first."))
		return b.String()
	}

	if !m.settings.KeepBinaries {
		b.WriteString(styles.WarningText.Render("[!] Binaries not available"))
		b.WriteString("\n\n")
		b.WriteString(styles.SubtleText.Render("Enable 'Keep Binaries' in Settings, then re-run."))
		return b.String()
	}

	isMultiProcess := false
	if m.selectedPolicy >= 0 && m.selectedPolicy < len(m.policies) {
		mp := m.policies[m.selectedPolicy].Run.MultiProcess
		if mp != nil && mp.Enabled && len(mp.Executables) > 0 {
			isMultiProcess = true
		}
	}

	if m.isExecuting && !isMultiProcess {
		b.WriteString(m.spinner.View())
		b.WriteString(" Running...")
		b.WriteString("\n\n")
		b.WriteString(styles.WarningText.Render("Press Ctrl+K to cancel"))
		return b.String()
	}

	if isMultiProcess {
		mp := m.policies[m.selectedPolicy].Run.MultiProcess

		b.WriteString(styles.Subtle.Render("Multi-Process Mode"))
		b.WriteString(styles.SubtleText.Render(fmt.Sprintf(" (%d processes)", len(mp.Executables))))
		b.WriteString("\n\n")

		for _, proc := range mp.Executables {
			b.WriteString(fmt.Sprintf("  • %s (%s)", proc.Name, proc.SourceFile))
			if proc.StartDelayMs > 0 {
				b.WriteString(styles.SubtleText.Render(fmt.Sprintf(" [delay: %dms]", proc.StartDelayMs)))
			}
			b.WriteString("\n")
		}

		b.WriteString("\n")
		if m.runInputFocused == 0 {
			b.WriteString(styles.Highlight.Render("> "))
			b.WriteString(styles.SelectedItem.Render("[ Run ]"))
		} else {
			b.WriteString("  ")
			b.WriteString(styles.NormalItem.Render("[ Run ]"))
		}
		b.WriteString("\n")

		if len(mp.TestScenarios) > 0 {
			b.WriteString("\n")
			b.WriteString(styles.Subtle.Render("Test Scenarios"))
			b.WriteString(styles.SubtleText.Render(fmt.Sprintf(" (%d)", len(mp.TestScenarios))))
			b.WriteString("\n\n")

			for i, scenario := range mp.TestScenarios {
				cursor := "  "
				style := styles.NormalItem
				if m.runInputFocused == 1+i {
					cursor = styles.Highlight.Render("> ")
					style = styles.SelectedItem
				}
				b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(scenario.Name)))
			}
		}

		if m.showMultiProcess && m.multiProcessResult != nil {
			b.WriteString("\n")
			b.WriteString(m.renderMultiProcessGrid())
		}
	} else {
		b.WriteString(styles.Subtle.Render("Custom Execution"))
		b.WriteString("\n\n")

		argsLabel := "  Arguments: "
		if m.runInputFocused == 0 {
			argsLabel = styles.Highlight.Render("> ") + "Arguments: "
		}
		b.WriteString(argsLabel)
		b.WriteString(m.runArgsInput.View())
		b.WriteString("\n")

		stdinLabel := "  Stdin:     "
		if m.runInputFocused == 1 {
			stdinLabel = styles.Highlight.Render("> ") + "Stdin:     "
		}
		b.WriteString(stdinLabel)
		b.WriteString(m.runStdinInput.View())
		b.WriteString("\n\n")

		if m.runInputFocused == 2 {
			b.WriteString(styles.Highlight.Render("> "))
			b.WriteString(styles.SelectedItem.Render("[ Run ]"))
		} else {
			b.WriteString("  ")
			b.WriteString(styles.SubtleText.Render("[ Run ]"))
		}
		b.WriteString("\n")

		if m.selectedPolicy >= 0 && m.selectedPolicy < len(m.policies) {
			testCases := m.policies[m.selectedPolicy].Run.TestCases
			if len(testCases) > 0 {
				b.WriteString("\n")
				b.WriteString(styles.Subtle.Render("Preset Test Cases"))
				b.WriteString(styles.SubtleText.Render(fmt.Sprintf(" (%d)", len(testCases))))
				b.WriteString("\n\n")

				for i, tc := range testCases {
					cursor := "  "
					style := styles.NormalItem
					if m.runInputFocused == 3+i {
						cursor = styles.Highlight.Render("> ")
						style = styles.SelectedItem
					}

					name := tc.Name
					if name == "" {
						name = fmt.Sprintf("Test %d", i+1)
					}

					argsInfo := ""
					if len(tc.Args) > 0 {
						argsInfo = fmt.Sprintf(" [%s]", strings.Join(tc.Args, " "))
					}

					b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, style.Render(name), styles.SubtleText.Render(argsInfo)))
				}
			}
		}

		if m.runResult != nil {
			b.WriteString("\n")
			b.WriteString(styles.Subtle.Render("─── Last Result ───"))
			b.WriteString("\n\n")
			b.WriteString(m.renderExecuteResult(*m.runResult))
		}

		if len(m.runTestResults) > 0 {
			b.WriteString("\n")
			b.WriteString(styles.Subtle.Render("─── Test Results ───"))
			b.WriteString("\n\n")

			passed := 0
			for _, tr := range m.runTestResults {
				if tr.Passed {
					passed++
				}
			}

			if passed == len(m.runTestResults) {
				b.WriteString(styles.SuccessText.Render(fmt.Sprintf("All %d tests passed!", passed)))
			} else {
				b.WriteString(styles.WarningText.Render(fmt.Sprintf("%d/%d tests passed", passed, len(m.runTestResults))))
			}
			b.WriteString("\n\n")

			for _, tr := range m.runTestResults {
				name := tr.TestCaseName
				if name == "" {
					name = "Test"
				}
				if tr.Passed {
					b.WriteString(styles.SuccessText.Render(fmt.Sprintf("  [PASS] %s", name)))
				} else {
					b.WriteString(styles.ErrorText.Render(fmt.Sprintf("  [FAIL] %s", name)))
				}
				b.WriteString(styles.SubtleText.Render(fmt.Sprintf(" (exit %d, %dms)", tr.ExitCode, tr.Duration.Milliseconds())))
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

func (m Model) renderMultiProcessGrid() string {
	if m.multiProcessResult == nil {
		return ""
	}

	var b strings.Builder

	if m.multiProcessResult.ScenarioName != "" {
		b.WriteString(styles.Subtle.Render(fmt.Sprintf("─── %s ───", m.multiProcessResult.ScenarioName)))
	} else {
		b.WriteString(styles.Subtle.Render("─── Multi-Process Results ───"))
	}
	b.WriteString("\n")
	b.WriteString(styles.SubtleText.Render(fmt.Sprintf("Total: %dms", m.multiProcessResult.TotalDuration.Milliseconds())))

	anyRunning := false
	anyKilled := false
	for _, pr := range m.multiProcessResult.Processes {
		if pr.Running {
			anyRunning = true
		}
		if pr.Killed {
			anyKilled = true
		}
	}

	if anyRunning {
		b.WriteString(styles.PrimaryText.Render(" [RUNNING...]"))
		b.WriteString(styles.SubtleText.Render(" (Ctrl+K to kill)"))
	} else if m.multiProcessResult.AllPassed {
		b.WriteString(styles.SuccessText.Render(" [ALL PASSED]"))
	} else if anyKilled {
		b.WriteString(styles.WarningText.Render(" [KILLED]"))
	} else if m.multiProcessResult.AllCompleted {
		b.WriteString(styles.WarningText.Render(" [Some failed]"))
	} else {
		b.WriteString(styles.ErrorText.Render(" [Incomplete]"))
	}
	b.WriteString("\n\n")

	processes := m.multiProcessResult.Order
	numProcs := len(processes)

	scenarioCount := 0
	if m.selectedPolicy >= 0 && m.selectedPolicy < len(m.policies) {
		mp := m.policies[m.selectedPolicy].Run.MultiProcess
		if mp != nil {
			scenarioCount = len(mp.TestScenarios)
		}
	}
	processStartIdx := 1 + scenarioCount

	availableWidth := m.width - 14
	if availableWidth < 40 {
		availableWidth = 40
	}

	minColWidth := 38
	useTwoColumns := availableWidth >= (minColWidth*2 + 4)

	if useTwoColumns {
		colWidth := (availableWidth - 4) / 2

		for i := 0; i < numProcs; i += 2 {
			row := m.renderProcessRow(processes, i, colWidth, true, scenarioCount)
			b.WriteString(row)
			if i+2 < numProcs {
				b.WriteString("\n")
			}
		}
	} else {
		colWidth := availableWidth

		for i := 0; i < numProcs; i++ {
			procName := processes[i]
			proc := m.multiProcessResult.Processes[procName]
			isSelected := m.runInputFocused == processStartIdx+i
			isFocused := m.selectedProcessIdx == i
			scrollOffset := 0
			if isFocused {
				scrollOffset = m.outputScroll
			}
			b.WriteString(m.renderProcessBox(proc, colWidth, isSelected, isFocused, scrollOffset))
			if i < numProcs-1 {
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

func (m Model) renderProcessRow(processes []string, startIdx, colWidth int, twoCol bool, scenarioCount int) string {
	if !twoCol || startIdx >= len(processes) {
		return ""
	}

	processStartIdx := 1 + scenarioCount

	procName := processes[startIdx]
	proc := m.multiProcessResult.Processes[procName]
	isSelected1 := m.runInputFocused == processStartIdx+startIdx
	isFocused1 := m.selectedProcessIdx == startIdx
	scroll1 := 0
	if isFocused1 {
		scroll1 = m.outputScroll
	}
	box1 := m.renderProcessBox(proc, colWidth, isSelected1, isFocused1, scroll1)

	if startIdx+1 >= len(processes) {
		return box1
	}

	procName2 := processes[startIdx+1]
	proc2 := m.multiProcessResult.Processes[procName2]
	isSelected2 := m.runInputFocused == processStartIdx+startIdx+1
	isFocused2 := m.selectedProcessIdx == startIdx+1
	scroll2 := 0
	if isFocused2 {
		scroll2 = m.outputScroll
	}
	box2 := m.renderProcessBox(proc2, colWidth, isSelected2, isFocused2, scroll2)

	return lipgloss.JoinHorizontal(lipgloss.Top, box1, "  ", box2)
}

func (m Model) renderProcessBox(proc *domain.ProcessResult, width int, isSelected, isFocused bool, scrollOffset int) string {
	borderColor := styles.Muted
	if isFocused {
		borderColor = styles.Accent
	} else if isSelected {
		borderColor = styles.PrimaryGlow
	} else if proc.Running {
		borderColor = styles.Primary
	} else if proc.Killed {
		borderColor = styles.Warning
	} else if proc.ExpectedExit != nil {
		if proc.Passed {
			borderColor = styles.Success
		} else {
			borderColor = styles.Error
		}
	}

	contentWidth := width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	var content strings.Builder

	allOutput := components.SanitizeDisplay(proc.Stdout)
	if proc.Stderr != "" {
		if allOutput != "" {
			allOutput += "\n" + styles.WarningText.Render("stderr:") + "\n" + components.SanitizeDisplay(proc.Stderr)
		} else {
			allOutput = styles.WarningText.Render("stderr:") + "\n" + components.SanitizeDisplay(proc.Stderr)
		}
	}

	wrappedLines := components.WrapLines(allOutput, contentWidth)
	maxShow := 8
	minContentLines := maxShow
	totalLines := len(wrappedLines)
	startIdx, endIdx := components.ScrollIndices(totalLines, maxShow, scrollOffset)

	header := proc.Name
	sourceInfo := fmt.Sprintf(" (%s)", proc.SourceFile)
	if len(header)+len(sourceInfo) > contentWidth-10 {
		maxSource := contentWidth - len(header) - 15
		if maxSource > 3 {
			sourceInfo = fmt.Sprintf(" (%s...)", proc.SourceFile[:maxSource])
		} else {
			sourceInfo = ""
		}
	}
	content.WriteString(styles.Subtle.Render(header))
	content.WriteString(styles.SubtleText.Render(sourceInfo))
	if isFocused && totalLines > maxShow {
		content.WriteString(styles.SubtleText.Render(fmt.Sprintf(" [%d-%d/%d]", startIdx+1, endIdx, totalLines)))
	}
	content.WriteString("\n")

	content.WriteString(renderProcessStatusLine(proc))
	content.WriteString("\n")

	var outputLines []string
	if proc.OutputMatch == domain.OutputMatchFail && len(proc.OutputDiff) > 0 {
		outputLines = append(outputLines, renderDiffLines(proc.OutputDiff, contentWidth)...)
		if proc.Stderr != "" {
			outputLines = appendStderrBlock(outputLines, proc.Stderr, contentWidth)
		}
	} else {
		outputLines = wrappedLines
	}

	totalLines = len(outputLines)
	startIdx, endIdx = components.ScrollIndices(totalLines, maxShow, scrollOffset)

	for i := startIdx; i < endIdx; i++ {
		content.WriteString(outputLines[i])
		content.WriteString("\n")
	}

	currentLines := endIdx - startIdx
	for i := currentLines; i < minContentLines; i++ {
		content.WriteString("\n")
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width)

	return box.Render(content.String())
}

func (m Model) renderExecuteResult(r domain.ExecuteResult) string {
	boxWidth := m.width - 14
	if boxWidth < 40 {
		boxWidth = 40
	}
	contentWidth := boxWidth - 4

	testCaseCount := 0
	if m.selectedPolicy >= 0 && m.selectedPolicy < len(m.policies) {
		testCaseCount = len(m.policies[m.selectedPolicy].Run.TestCases)
	}
	outputBoxIdx := 3 + testCaseCount
	isSelected := m.runInputFocused == outputBoxIdx
	isFocused := m.selectedProcessIdx >= 0

	borderColor := styles.Muted
	if isFocused {
		borderColor = styles.Accent
	} else if isSelected {
		borderColor = styles.PrimaryGlow
	} else if r.ExitCode == 0 && !r.TimedOut {
		borderColor = styles.Success
	} else {
		borderColor = styles.Warning
	}

	var content strings.Builder

	content.WriteString(renderExecuteStatusLine(r))

	var outputLines []string
	if r.OutputMatch == domain.OutputMatchFail && len(r.OutputDiff) > 0 {
		outputLines = append(outputLines, renderDiffLines(r.OutputDiff, contentWidth)...)
		if r.Stderr != "" {
			outputLines = appendStderrBlock(outputLines, r.Stderr, contentWidth)
		}
	} else {
		allOutput := components.SanitizeDisplay(r.Stdout)
		if r.Stderr != "" {
			if allOutput != "" {
				allOutput += "\n" + styles.WarningText.Render("stderr:") + "\n" + components.SanitizeDisplay(r.Stderr)
			} else {
				allOutput = styles.WarningText.Render("stderr:") + "\n" + components.SanitizeDisplay(r.Stderr)
			}
		}
		outputLines = components.WrapLines(allOutput, contentWidth)
	}

	maxShow := 15
	totalLines := len(outputLines)
	startIdx, endIdx := components.ScrollIndices(totalLines, maxShow, m.outputScroll)

	if isFocused && totalLines > maxShow {
		content.WriteString(styles.SubtleText.Render(fmt.Sprintf(" [%d-%d/%d]", startIdx+1, endIdx, totalLines)))
	}
	content.WriteString("\n")

	if totalLines > 0 {
		for i := startIdx; i < endIdx; i++ {
			content.WriteString(outputLines[i])
			content.WriteString("\n")
		}
		linesShown := endIdx - startIdx
		for i := linesShown; i < maxShow; i++ {
			content.WriteString("\n")
		}
	} else {
		content.WriteString(styles.SubtleText.Render("(no output)\n"))
		for i := 1; i < maxShow; i++ {
			content.WriteString("\n")
		}
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(boxWidth)

	return box.Render(content.String())
}

func renderOutputMatchLabel(status domain.OutputMatchStatus, diffCount int, diffSuffix string) string {
	switch status {
	case domain.OutputMatchPass:
		return styles.SuccessText.Render(" | Output: PASS")
	case domain.OutputMatchFail:
		return styles.WarningText.Render(fmt.Sprintf(" | Output: CHECK (%d%s)", diffCount, diffSuffix))
	case domain.OutputMatchMissing:
		return styles.ErrorText.Render(" | Output: MISSING")
	default:
		return ""
	}
}

func renderProcessStatusLine(proc *domain.ProcessResult) string {
	var b strings.Builder
	switch {
	case proc.Running:
		b.WriteString(styles.Highlight.Render("[RUNNING]"))
		b.WriteString(styles.SubtleText.Render(" ..."))
	case proc.Killed:
		b.WriteString(styles.WarningText.Render("[KILLED]"))
		b.WriteString(styles.SubtleText.Render(fmt.Sprintf(" %dms", proc.Duration.Milliseconds())))
	case proc.TimedOut:
		b.WriteString(styles.ErrorText.Render("[TIMEOUT]"))
		b.WriteString(styles.SubtleText.Render(fmt.Sprintf(" %dms", proc.Duration.Milliseconds())))
	case proc.ExpectedExit != nil:
		if proc.Passed {
			b.WriteString(styles.SuccessText.Render(fmt.Sprintf("[PASS] exit %d", proc.ExitCode)))
		} else {
			b.WriteString(styles.ErrorText.Render(fmt.Sprintf("[FAIL] exit %d (expected %d)", proc.ExitCode, *proc.ExpectedExit)))
		}
		b.WriteString(styles.SubtleText.Render(fmt.Sprintf(" %dms", proc.Duration.Milliseconds())))
	case proc.ExitCode == 0:
		b.WriteString(styles.SuccessText.Render(fmt.Sprintf("[OK] exit %d", proc.ExitCode)))
		b.WriteString(styles.SubtleText.Render(fmt.Sprintf(" %dms", proc.Duration.Milliseconds())))
	default:
		b.WriteString(styles.WarningText.Render(fmt.Sprintf("[EXIT %d]", proc.ExitCode)))
		b.WriteString(styles.SubtleText.Render(fmt.Sprintf(" %dms", proc.Duration.Milliseconds())))
	}
	b.WriteString(renderOutputMatchLabel(proc.OutputMatch, len(proc.OutputDiff), ""))
	return b.String()
}

func renderExecuteStatusLine(r domain.ExecuteResult) string {
	var b strings.Builder
	if r.TimedOut {
		b.WriteString(styles.ErrorText.Render("[TIMEOUT] Execution timed out"))
	} else if r.ExitCode == 0 {
		b.WriteString(styles.SuccessText.Render(fmt.Sprintf("[OK] exit %d", r.ExitCode)))
	} else {
		b.WriteString(styles.WarningText.Render(fmt.Sprintf("[EXIT %d]", r.ExitCode)))
	}
	b.WriteString(styles.SubtleText.Render(fmt.Sprintf(" %dms", r.Duration.Milliseconds())))
	b.WriteString(renderOutputMatchLabel(r.OutputMatch, len(r.OutputDiff), " diffs"))
	return b.String()
}

func renderDiffLines(diff []domain.DiffLine, contentWidth int) []string {
	if len(diff) == 0 {
		return nil
	}
	lineStyle := lipgloss.NewStyle().Width(contentWidth).MaxWidth(contentWidth)
	lines := make([]string, 0, len(diff))
	for _, d := range diff {
		var prefix string
		switch d.Type {
		case "removed":
			prefix = "- "
		case "added":
			prefix = "+ "
		default:
			prefix = "  "
		}
		plain := prefix + components.SanitizeDisplay(d.Content)
		plain = components.TruncateToWidth(plain, contentWidth)
		switch d.Type {
		case "removed":
			lines = append(lines, styles.ErrorText.Render(lineStyle.Render(plain)))
		case "added":
			lines = append(lines, styles.SuccessText.Render(lineStyle.Render(plain)))
		default:
			lines = append(lines, lineStyle.Render(plain))
		}
	}
	return lines
}

func appendStderrBlock(lines []string, stderr string, contentWidth int) []string {
	lines = append(lines, "")
	lineStyle := lipgloss.NewStyle().Width(contentWidth).MaxWidth(contentWidth)
	lines = append(lines, styles.WarningText.Render(lineStyle.Render("─── stderr ───")))
	lines = append(lines, components.WrapLines(components.SanitizeDisplay(stderr), contentWidth)...)
	return lines
}

func (m Model) renderExport() string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("Export Results"))
	b.WriteString("\n\n")

	boxWidth := m.width - 8
	if boxWidth < 60 {
		boxWidth = 60
	}

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
