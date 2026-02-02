package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/feli05/autoscan/internal/domain"
	"github.com/feli05/autoscan/internal/tui/components"
	"github.com/feli05/autoscan/internal/tui/styles"
	"github.com/feli05/autoscan/internal/tui/views/banned"
	"github.com/feli05/autoscan/internal/tui/views/directory"
	exportview "github.com/feli05/autoscan/internal/tui/views/export"
	"github.com/feli05/autoscan/internal/tui/views/home"
	policyview "github.com/feli05/autoscan/internal/tui/views/policy"
	"github.com/feli05/autoscan/internal/tui/views/settings"
	"github.com/feli05/autoscan/internal/tui/views/submissions"
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
		content = policyview.SelectView(policyview.SelectState{
			Policies:       m.policies,
			SelectedPolicy: m.selectedPolicy,
			InputError:     m.inputError,
			Width:          m.width,
		})
	case ViewPolicyManage:
		content = policyview.ManageView(policyview.ManageState{
			Policies:           m.policies,
			PolicyManageCursor: m.policyManageCursor,
			ConfirmDelete:      m.confirmDelete,
			Width:              m.width,
		})
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
		content = banned.View(banned.State{
			Width:            m.width,
			BannedList:       m.bannedList,
			BannedCursorEdit: m.bannedCursorEdit,
			BannedEditing:    m.bannedEditing,
			BannedInput:      m.bannedInput,
		})
	case ViewSettings:
		content = settings.View(settings.State{
			Settings:       &m.settings,
			SettingsCursor: m.settingsCursor,
			Width:          m.width,
		})
	case ViewDirectoryInput:
		content = directory.View(directory.State{
			Width:         m.width,
			InputError:    m.inputError,
			FolderBrowser: m.folderBrowser,
		})
	case ViewSubmissions:
		content = submissions.View(m.buildSubmissionsState())
	case ViewDetails:
		content = m.renderDetails()
	case ViewExport:
		content = exportview.View(exportview.State{
			Width:        m.width,
			ExportCursor: m.exportCursor,
			Report:       m.report,
		})
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
