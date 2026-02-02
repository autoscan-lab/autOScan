package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/feli05/autoscan/internal/config"
	"github.com/feli05/autoscan/internal/domain"
	"github.com/feli05/autoscan/internal/engine"
	"github.com/feli05/autoscan/internal/policy"
	"github.com/feli05/autoscan/internal/tui/components"
	"github.com/feli05/autoscan/internal/tui/views/home"
	"github.com/feli05/autoscan/internal/tui/views/settings"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		switch m.currentView {
		case ViewHome:
			result := home.Update(home.State{
				Width:         m.width,
				MenuItem:      int(m.menuItem),
				ConfirmDelete: m.confirmDelete,
				PolicyCount:   len(m.policies),
			}, msg)
			m.menuItem = MenuItem(result.MenuItem)
			m.confirmDelete = result.ConfirmDelete
			if result.ResetPolicyManageCursor {
				m.policyManageCursor = 0
			}
			if result.ResetSettingsCursor {
				m.settingsCursor = 0
			}
			switch result.Navigation {
			case home.NavPolicySelect:
				m.currentView = ViewPolicySelect
			case home.NavPolicyManage:
				m.currentView = ViewPolicyManage
			case home.NavSettings:
				m.currentView = ViewSettings
			case home.NavQuit:
				return m, tea.Quit
			case home.NavUninstall:
				return m, m.doUninstall()
			}
			return m, nil
		case ViewPolicySelect:
			return m.updatePolicySelect(msg)
		case ViewPolicyManage:
			return m.updatePolicyManage(msg)
		case ViewPolicyEditor:
			return m.updatePolicyEditor(msg)
		case ViewBannedEditor:
			return m.updateBannedEditor(msg)
		case ViewSettings:
			result := settings.Update(settings.State{
				Settings:       &m.settings,
				SettingsCursor: m.settingsCursor,
				Width:          m.width,
			}, msg)
			m.settings = result.Settings
			m.settingsCursor = result.SettingsCursor
			if result.GoBack {
				m.currentView = ViewHome
			}
			return m, nil
		case ViewDirectoryInput:
			return m.updateDirectoryInput(msg)
		case ViewSubmissions:
			return m.updateSubmissions(msg)
		case ViewDetails:
			return m.updateDetails(msg)
		case ViewExport:
			return m.updateExport(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.width < minWidth {
			m.width = minWidth
		}
		if m.height < minHeight {
			m.height = minHeight
		}
		m.visibleRows = m.height - 12
		if m.visibleRows < 5 {
			m.visibleRows = 5
		}
		m.helpPanel.SetWidth(min(28, m.width/4))
		m.policyEditor.SetWidth(m.width)

	case components.AnimationTickMsg:
		cmd := m.eyeAnimation.Update(msg)
		cmds = append(cmds, cmd)

	case policiesLoadedMsg:
		m.policies = msg
		m.statusMsg = ""
		m.helpPanel.SetPolicyCount(len(msg))

	case runCompleteMsg:
		report := domain.RunReport(msg)
		m.report = &report
		m.results = m.report.Results
		m.isRunning = false
		m.completed = len(m.results)
		m.runError = ""

		m.submissionsTab = 0
		m.similarityPairsByProcess = make(map[string][]SimilarityPair)
		m.similarityStateByProcess = make(map[string]SimilarityComputeState)
		m.similarityErrorByProcess = make(map[string]string)
		m.similarityInFlight = make(map[string]bool)
		m.similarityCursor = 0
		m.similarityScroll = 0
		m.initSimilarityProcesses()

		m.resetPairDetailState()

	case errorMsg:
		m.runError = msg.Error()
		m.isRunning = false

	case exportDoneMsg:
		m.statusMsg = fmt.Sprintf("Exported to %s", msg.path)

	case similarityStartedMsg:
		if msg.runID == m.runID {
			m.similarityStateByProcess[msg.process] = SimilarityComputing
			delete(m.similarityErrorByProcess, msg.process)
		}

	case similarityComputedMsg:
		if msg.runID == m.runID {
			m.similarityPairsByProcess[msg.process] = msg.pairs
			m.similarityStateByProcess[msg.process] = SimilarityDone
			delete(m.similarityErrorByProcess, msg.process)
			delete(m.similarityInFlight, msg.process)
		}

	case similarityErrorMsg:
		if msg.runID == m.runID {
			m.similarityStateByProcess[msg.process] = SimilarityError
			m.similarityErrorByProcess[msg.process] = msg.err.Error()
			delete(m.similarityInFlight, msg.process)
		}

	case pairDetailLoadedMsg:
		if msg.runID == m.runID && msg.process == m.pairDetailProcess && msg.pairIndex == m.pairDetailPairIndex {
			m.pairDetailContentA = msg.contentA
			m.pairDetailContentB = msg.contentB
			if msg.err != nil {
				m.pairDetailLoadErr = msg.err.Error()
			} else {
				m.pairDetailLoadErr = ""
			}
		}

	case policySavedMsg:
		m.currentView = ViewPolicyManage
		m.statusMsg = fmt.Sprintf("Policy saved to %s", msg.path)
		return m, m.loadPolicies()

	case policySaveErrorMsg:
		m.policyEditor.errorMsg = msg.err

	case policyDeletedMsg:
		m.currentView = ViewPolicyManage
		m.statusMsg = fmt.Sprintf("Deleted policy: %s", msg.name)
		m.confirmDelete = false
		return m, m.loadPolicies()

	case uninstallDoneMsg:
		fmt.Println("\nautoscan has been uninstalled.")
		fmt.Println("Config removed from ~/.config/autoscan/")
		fmt.Println("Binary removed from ~/.local/bin/autoscan")
		return m, tea.Quit

	case bannedListLoadedMsg:
		m.bannedList = []string(msg)
		m.bannedCursorEdit = 0
		m.currentView = ViewBannedEditor
		return m, nil

	case bannedListSavedMsg:
		m.statusMsg = "Banned list saved"
		return m, nil

	case executeResultMsg:
		m.runResult = &msg.result
		m.isExecuting = false
		m.selectedProcessIdx = -1
		m.outputScroll = 0
		return m, nil

	case executeTestResultsMsg:
		m.runTestResults = msg.results
		m.isExecuting = false
		return m, nil

	case multiProcessResultMsg:
		if msg.result != nil {
			m.multiProcessResult = msg.result
		}
		m.isExecuting = false
		m.showMultiProcess = true
		m.multiProcessUpdateChan = nil
		return m, nil

	case multiProcessUpdateMsg:
		m.multiProcessResult = msg.result
		m.showMultiProcess = true
		if m.multiProcessUpdateChan != nil {
			return m, waitForMultiProcessUpdates(m.multiProcessUpdateChan)
		}
		return m, nil
	}

	var spinnerCmd tea.Cmd
	m.spinner, spinnerCmd = m.spinner.Update(msg)
	cmds = append(cmds, spinnerCmd)

	if m.currentView == ViewPolicyEditor {
		cmd := m.policyEditor.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.currentView == ViewSubmissions && m.searchActive {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) updatePolicySelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.selectedPolicy < len(m.policies)-1 {
			m.selectedPolicy++
			m.executor = nil
			m.inputError = ""
		}
	case "k", "up":
		if m.selectedPolicy > 0 {
			m.selectedPolicy--
			m.executor = nil
			m.inputError = ""
		}
	case "enter":
		if len(m.policies) > 0 {
			if m.selectedPolicy < len(m.policies) {
				p := m.policies[m.selectedPolicy]
				if p != nil {
					isMulti := p.Run.MultiProcess != nil && p.Run.MultiProcess.Enabled
					if isMulti {
						if len(p.Run.MultiProcess.Executables) == 0 {
							m.inputError = "Multi-process policy needs at least one executable"
							return m, nil
						}
						for _, proc := range p.Run.MultiProcess.Executables {
							if strings.TrimSpace(proc.SourceFile) == "" {
								m.inputError = "Multi-process policy has a missing source file"
								return m, nil
							}
						}
					} else if strings.TrimSpace(p.Compile.SourceFile) == "" {
						m.inputError = "Single-process policy requires source_file"
						return m, nil
					}
				}
			}
			m.inputError = ""
			m.currentView = ViewDirectoryInput
			m.folderBrowser.Reset(m.root)
			return m, nil
		}
	case "q", "esc":
		m.inputError = ""
		m.currentView = ViewHome
	}
	return m, nil
}

func (m Model) updatePolicyManage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	maxCursor := len(m.policies)

	switch msg.String() {
	case "j", "down":
		if m.policyManageCursor < maxCursor {
			m.policyManageCursor++
		}
	case "k", "up":
		if m.policyManageCursor > -1 {
			m.policyManageCursor--
		}
	case "enter":
		if m.policyManageCursor == -1 {
			return m, m.loadBannedList()
		} else if m.policyManageCursor == 0 {
			return m, m.openPolicyEditor(nil)
		} else {
			return m, m.openPolicyEditor(m.policies[m.policyManageCursor-1])
		}
	case "e":
		if m.policyManageCursor > 0 && m.policyManageCursor <= len(m.policies) {
			return m, m.openPolicyEditor(m.policies[m.policyManageCursor-1])
		}
	case "d":
		if m.policyManageCursor > 0 && m.policyManageCursor <= len(m.policies) {
			m.confirmDelete = true
		}
	case "y":
		if m.confirmDelete && m.policyManageCursor > 0 && m.policyManageCursor <= len(m.policies) {
			return m, DeletePolicy(m.policies[m.policyManageCursor-1])
		}
	case "n":
		m.confirmDelete = false
	case "q", "esc":
		if m.confirmDelete {
			m.confirmDelete = false
		} else {
			m.currentView = ViewHome
		}
	}
	return m, nil
}

func (m Model) updatePolicyEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" && !m.policyEditor.InSubMode() {
		m.currentView = ViewPolicyManage
		m.policyEditor.errorMsg = ""
		return m, nil
	}

	cmd := m.policyEditor.Update(msg)
	return m, cmd
}

func (m Model) updateDirectoryInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.currentView = ViewPolicySelect
		m.inputError = ""
		return m, nil
	}

	selected, cmd := m.folderBrowser.Update(msg)
	if selected {
		m.root = m.folderBrowser.Selected()
		m.inputError = ""
		return m.startRun()
	}

	return m, cmd
}

func (m Model) updateSubmissions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.isRunning {
		return m, nil
	}

	if m.pairDetailOpen {
		return m.updateSubmissionsPairDetail(msg)
	}

	if msg.String() == "tab" {
		if m.report != nil {
			m.submissionsTab = (m.submissionsTab + 1) % 2
			if m.submissionsTab == 0 {
				m.cursor = 0
				m.scrollOffset = 0
			} else {
				m.similarityCursor = 0
				m.similarityScroll = 0
				proc := m.currentSimilarityProcessName()
				if proc != "" && m.similarityStateByProcess[proc] == SimilarityNotStarted {
					return m, m.computeSimilarityForProcess(proc)
				}
			}
		}
		return m, nil
	}

	if m.submissionsTab == 1 {
		return m.updateSubmissionsSimilarity(msg)
	}

	if m.searchActive {
		switch msg.String() {
		case "esc", "down", "j":
			m.searchActive = false
			m.searchQuery = m.searchInput.Value()
			m.searchInput.Blur()
			m.cursor = 0
			m.scrollOffset = 0
			return m, nil
		case "enter":
			m.searchActive = false
			m.searchQuery = m.searchInput.Value()
			m.searchInput.Blur()
			return m, nil
		}

		prev := m.searchInput.Value()
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		if m.searchInput.Value() != prev {
			m.searchQuery = m.searchInput.Value()
			m.cursor = 0
			m.scrollOffset = 0
			m.clearRunResults()
		}
		return m, cmd
	}

	filtered := m.filteredResults()

	switch msg.String() {
	case "/":
		m.searchActive = true
		m.searchInput.Focus()
		return m, textinput.Blink
	case "esc":
		if strings.TrimSpace(m.searchQuery) != "" {
			m.searchQuery = ""
			m.searchInput.SetValue("")
			m.cursor = 0
			m.scrollOffset = 0
			m.clearRunResults()
			return m, nil
		}
		m.currentView = ViewHome
		m.results = nil
		m.report = nil
		return m, nil
	case "j", "down":
		if m.cursor < len(filtered)-1 {
			m.cursor++
			if m.cursor >= m.scrollOffset+m.visibleRows {
				m.scrollOffset++
			}
			m.clearRunResults()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.scrollOffset {
				m.scrollOffset--
			}
			m.clearRunResults()
		} else {
			m.searchActive = true
			m.searchInput.Focus()
			return m, textinput.Blink
		}
	case "enter":
		if len(filtered) > 0 {
			m.currentView = ViewDetails
			m.detailsTab = 0
			m.detailScroll = 0
			m.clearRunResults()
			m.executor = nil
		}
	case "f":
		m.filter = (m.filter + 1) % 4
		m.cursor = 0
		m.scrollOffset = 0
	case "r":
		return m.startRun()
	case "e":
		m.currentView = ViewExport
		m.exportCursor = 0
	case "q":
		m.currentView = ViewHome
		m.results = nil
		m.report = nil
	}

	return m, nil
}

func (m Model) updateSubmissionsSimilarity(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.currentView = ViewHome
		m.results = nil
		m.report = nil
		m.resetPairDetailState()
		m.similarityPairsByProcess = make(map[string][]SimilarityPair)
		m.similarityStateByProcess = make(map[string]SimilarityComputeState)
		m.similarityErrorByProcess = make(map[string]string)
		return m, nil
	}

	if len(m.similarityProcessNames) == 0 {
		return m, nil
	}

	currentProc := m.currentSimilarityProcessName()
	if currentProc != "" && m.similarityStateByProcess[currentProc] == SimilarityNotStarted {
		return m, m.computeSimilarityForProcess(currentProc)
	}

	pairs := m.similarityPairsByProcess[currentProc]
	dataRows := min(30, m.visibleRows-1)
	if dataRows < 6 {
		dataRows = 6
	}

	switch msg.String() {
	case "j", "down":
		if len(pairs) == 0 {
			return m, nil
		}
		if m.similarityCursor < len(pairs)-1 {
			m.similarityCursor++
			if m.similarityCursor >= m.similarityScroll+dataRows {
				m.similarityScroll++
			}
		}
	case "k", "up":
		if len(pairs) == 0 {
			return m, nil
		}
		if m.similarityCursor > 0 {
			m.similarityCursor--
			if m.similarityCursor < m.similarityScroll {
				m.similarityScroll--
			}
		}
	case "l", "right":
		if m.similaritySelectedProc < len(m.similarityProcessNames)-1 {
			m.similaritySelectedProc++
			m.similarityCursor = 0
			m.similarityScroll = 0
			proc := m.currentSimilarityProcessName()
			if proc != "" && m.similarityStateByProcess[proc] == SimilarityNotStarted {
				return m, m.computeSimilarityForProcess(proc)
			}
		}
	case "h", "left":
		if m.similaritySelectedProc > 0 {
			m.similaritySelectedProc--
			m.similarityCursor = 0
			m.similarityScroll = 0
			proc := m.currentSimilarityProcessName()
			if proc != "" && m.similarityStateByProcess[proc] == SimilarityNotStarted {
				return m, m.computeSimilarityForProcess(proc)
			}
		}
	case "enter":
		if len(pairs) == 0 || m.similarityCursor >= len(pairs) {
			return m, nil
		}
		pair := pairs[m.similarityCursor]
		srcFile := m.resolveSimilaritySourceFile(currentProc)
		if srcFile == "" && len(m.results) > 0 && len(m.results[0].Submission.CFiles) > 0 {
			srcFile = m.results[0].Submission.CFiles[0]
		}
		if srcFile == "" {
			return m, nil
		}
		pathA := filepath.Join(m.results[pair.AIndex].Submission.Path, srcFile)
		pathB := filepath.Join(m.results[pair.BIndex].Submission.Path, srcFile)
		m.resetPairDetailViewState()
		m.pairDetailOpen = true
		m.pairDetailProcess = currentProc
		m.pairDetailPairIndex = m.similarityCursor
		return m, loadPairDetailFiles(currentProc, m.similarityCursor, pathA, pathB, m.runID)
	}
	return m, nil
}

func loadPairDetailFiles(process string, pairIndex int, pathA, pathB string, runID int64) tea.Cmd {
	return func() tea.Msg {
		contentA, errA := os.ReadFile(pathA)
		if errA != nil {
			return pairDetailLoadedMsg{process: process, pairIndex: pairIndex, contentA: nil, contentB: nil, err: errA, runID: runID}
		}
		contentB, errB := os.ReadFile(pathB)
		if errB != nil {
			return pairDetailLoadedMsg{process: process, pairIndex: pairIndex, contentA: contentA, contentB: nil, err: errB, runID: runID}
		}
		return pairDetailLoadedMsg{process: process, pairIndex: pairIndex, contentA: contentA, contentB: contentB, err: nil, runID: runID}
	}
}

func (m Model) updateSubmissionsPairDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	pairs := m.similarityPairsByProcess[m.pairDetailProcess]
	if m.pairDetailPairIndex >= len(pairs) {
		m.resetPairDetailState()
		return m, nil
	}

	const pairDetailMaxPaneHeight = 30
	paneHeight := pairDetailMaxPaneHeight
	if m.visibleRows < paneHeight {
		paneHeight = m.visibleRows
	}
	if paneHeight < 8 {
		paneHeight = 8
	}

	linesA := len(strings.Split(string(m.pairDetailContentA), "\n"))
	linesB := len(strings.Split(string(m.pairDetailContentB), "\n"))
	maxScrollA := max(0, linesA-paneHeight)
	maxScrollB := max(0, linesB-paneHeight)
	_, contentWidth := pairDetailPaneWidths(m.width)
	maxHScrollA := max(0, maxDisplayWidthForContent(m.pairDetailContentA)-contentWidth)
	maxHScrollB := max(0, maxDisplayWidthForContent(m.pairDetailContentB)-contentWidth)

	switch msg.String() {
	case "esc", "q":
		m.pairDetailOpen = false
		return m, nil
	case "h":
		m.pairDetailFocusedPane = 0
		return m, nil
	case "l":
		m.pairDetailFocusedPane = 1
		return m, nil
	case "down":
		if m.pairDetailFocusedPane == 0 {
			if m.pairDetailScrollA < maxScrollA {
				m.pairDetailScrollA++
			}
		} else {
			if m.pairDetailScrollB < maxScrollB {
				m.pairDetailScrollB++
			}
		}
	case "up":
		if m.pairDetailFocusedPane == 0 {
			if m.pairDetailScrollA > 0 {
				m.pairDetailScrollA--
			}
		} else {
			if m.pairDetailScrollB > 0 {
				m.pairDetailScrollB--
			}
		}
	case "right":
		if m.pairDetailFocusedPane == 0 {
			if m.pairDetailHScrollA < maxHScrollA {
				m.pairDetailHScrollA++
			}
		} else {
			if m.pairDetailHScrollB < maxHScrollB {
				m.pairDetailHScrollB++
			}
		}
	case "left":
		if m.pairDetailFocusedPane == 0 {
			if m.pairDetailHScrollA > 0 {
				m.pairDetailHScrollA--
			}
		} else {
			if m.pairDetailHScrollB > 0 {
				m.pairDetailHScrollB--
			}
		}
	}
	return m, nil
}

func maxDisplayWidthForContent(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	maxWidth := 0
	for _, line := range strings.Split(string(content), "\n") {
		line = components.SanitizeDisplay(line)
		line = expandTabsForPane(line, pairDetailTabWidth)
		if w := lipgloss.Width(line); w > maxWidth {
			maxWidth = w
		}
	}
	return maxWidth
}

func (m Model) updateDetails(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	getBannedFuncCount := func() int {
		filtered := m.filteredResults()
		if m.cursor >= len(filtered) {
			return 0
		}
		return len(filtered[m.cursor].Scan.HitsByFunction)
	}

	if m.detailsTab == 3 {
		return m.updateRunTab(msg)
	}

	switch msg.String() {
	case "tab":
		m.detailsTab = (m.detailsTab + 1) % 4
		m.detailScroll = 0
		m.bannedCursor = 0
		if m.detailsTab == 3 {
			m.runInputFocused = 0
			m.runArgsInput.Focus()
			m.runStdinInput.Blur()
		}
	case "shift+tab":
		m.detailsTab = (m.detailsTab + 3) % 4
		m.detailScroll = 0
		m.bannedCursor = 0
		if m.detailsTab == 3 {
			m.runInputFocused = 0
			m.runArgsInput.Focus()
			m.runStdinInput.Blur()
		}
	case "j", "down":
		if m.detailsTab == 1 {
			maxCursor := getBannedFuncCount() - 1
			if maxCursor >= 0 && m.bannedCursor < maxCursor {
				m.bannedCursor++
			}
		} else {
			m.detailScroll++
		}
	case "k", "up":
		if m.detailsTab == 1 {
			if m.bannedCursor > 0 {
				m.bannedCursor--
			}
		} else if m.detailScroll > 0 {
			m.detailScroll--
		}
	case "enter", " ":
		if m.detailsTab == 1 {
			if m.expandedFuncs == nil {
				m.expandedFuncs = make(map[string]bool)
			}
			filtered := m.filteredResults()
			if m.cursor < len(filtered) {
				r := filtered[m.cursor]
				var funcNames []string
				for fn := range r.Scan.HitsByFunction {
					funcNames = append(funcNames, fn)
				}
				sort.Strings(funcNames)
				if m.bannedCursor < len(funcNames) {
					fn := funcNames[m.bannedCursor]
					m.expandedFuncs[fn] = !m.expandedFuncs[fn]
				}
			}
		}
	case "q", "esc":
		m.currentView = ViewSubmissions
		m.expandedFuncs = nil
		m.bannedCursor = 0
		m.clearRunResults()
	}
	return m, nil
}

func (m Model) updateRunTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.isExecuting {
		switch msg.String() {
		case "ctrl+k", "K":
			if m.runCancelFunc != nil {
				m.runCancelFunc()
				m.runCancelFunc = nil
			}
			m.isExecuting = false
			m.statusMsg = "Processes killed (SIGKILL)"
			return m, nil
		}
		return m, nil
	}

	isMultiProcess := false
	var mp *policy.MultiProcessConfig
	if m.selectedPolicy >= 0 && m.selectedPolicy < len(m.policies) {
		mp = m.policies[m.selectedPolicy].Run.MultiProcess
		if mp != nil && mp.Enabled && len(mp.Executables) > 0 {
			isMultiProcess = true
		}
	}

	if isMultiProcess {
		scenarioCount := len(mp.TestScenarios)
		maxFocus := scenarioCount

		if m.multiProcessResult != nil && m.selectedProcessIdx >= 0 {
			numProcs := len(m.multiProcessResult.Order)
			maxScroll := 0
			if m.selectedProcessIdx < numProcs {
				procName := m.multiProcessResult.Order[m.selectedProcessIdx]
				proc := m.multiProcessResult.Processes[procName]

				boxWidth := (m.width - 20) / 2
				if boxWidth < 30 {
					boxWidth = 30
				}
				contentWidth := boxWidth - 4

				var outputLen int
				if proc.OutputMatch == domain.OutputMatchFail && len(proc.OutputDiff) > 0 {
					outputLen = len(proc.OutputDiff)
					if proc.Stderr != "" {
						outputLen++
						outputLen++
						outputLen += len(components.WrapLines(proc.Stderr, contentWidth))
					}
				} else {
					allOutput := proc.Stdout
					if proc.Stderr != "" {
						if allOutput != "" {
							allOutput += "\nstderr:\n" + proc.Stderr
						} else {
							allOutput = "stderr:\n" + proc.Stderr
						}
					}
					outputLen = len(components.WrapLines(allOutput, contentWidth))
				}
				maxScroll = outputLen - 8
				if maxScroll < 0 {
					maxScroll = 0
				}
			}

			switch msg.String() {
			case "up", "k":
				if m.outputScroll > 0 {
					m.outputScroll--
				}
				return m, nil
			case "down", "j":
				if m.outputScroll < maxScroll {
					m.outputScroll++
				}
				return m, nil
			case "esc", "enter":
				m.selectedProcessIdx = -1
				m.outputScroll = 0
				return m, nil
			}
			return m, nil
		}

		if m.multiProcessResult != nil && len(m.multiProcessResult.Order) > 0 {
			numProcs := len(m.multiProcessResult.Order)
			processStartIdx := 1 + scenarioCount

			switch msg.String() {
			case "tab":
				m.detailsTab = 0
				m.detailScroll = 0
				return m, nil

			case "shift+tab":
				m.detailsTab = 2
				m.detailScroll = 0
				return m, nil

			case "down", "j":
				maxIdx := processStartIdx + numProcs - 1
				if m.runInputFocused < maxIdx {
					m.runInputFocused++
				}
				return m, nil

			case "up", "k":
				if m.runInputFocused > 0 {
					m.runInputFocused--
				}
				return m, nil

			case "enter":
				if m.runInputFocused == 0 {
					return m, m.executeMultiProcess()
				} else if m.runInputFocused > 0 && m.runInputFocused <= scenarioCount {
					return m, m.executeMultiProcessScenario(m.runInputFocused - 1)
				} else if m.runInputFocused >= processStartIdx {
					m.selectedProcessIdx = m.runInputFocused - processStartIdx
					m.outputScroll = 0
					return m, nil
				}

			case "m":
				return m, m.executeMultiProcess()

			case "1", "2", "3", "4", "5", "6", "7", "8", "9":
				idx := int(msg.String()[0] - '1')
				if idx >= 0 && idx < scenarioCount {
					return m, m.executeMultiProcessScenario(idx)
				}

			case "esc", "q":
				m.currentView = ViewSubmissions
				m.expandedFuncs = nil
				m.multiProcessResult = nil
				m.showMultiProcess = false
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "tab":
			m.detailsTab = 0
			m.detailScroll = 0
			return m, nil

		case "shift+tab":
			m.detailsTab = 2
			m.detailScroll = 0
			return m, nil

		case "down", "j":
			if m.runInputFocused < maxFocus {
				m.runInputFocused++
			}
			return m, nil

		case "up", "k":
			if m.runInputFocused > 0 {
				m.runInputFocused--
			}
			return m, nil

		case "enter":
			if m.runInputFocused == 0 {
				return m, m.executeMultiProcess()
			} else if m.runInputFocused > 0 && m.runInputFocused <= scenarioCount {
				return m, m.executeMultiProcessScenario(m.runInputFocused - 1)
			}

		case "m":
			return m, m.executeMultiProcess()

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(msg.String()[0] - '1')
			if idx >= 0 && idx < scenarioCount {
				return m, m.executeMultiProcessScenario(idx)
			}

		case "esc", "q":
			m.currentView = ViewSubmissions
			m.expandedFuncs = nil
			m.multiProcessResult = nil
			m.showMultiProcess = false
			return m, nil
		}

		return m, nil
	}

	testCaseCount := 0
	if m.selectedPolicy >= 0 && m.selectedPolicy < len(m.policies) {
		testCaseCount = len(m.policies[m.selectedPolicy].Run.TestCases)
	}

	maxFocus := 2 + testCaseCount
	outputBoxIdx := maxFocus + 1

	if m.runResult != nil && m.selectedProcessIdx >= 0 {
		// Calculate content width same as view
		boxWidth := m.width - 14
		if boxWidth < 40 {
			boxWidth = 40
		}
		contentWidth := boxWidth - 4

		var outputLen int
		if m.runResult.OutputMatch == domain.OutputMatchFail && len(m.runResult.OutputDiff) > 0 {
			outputLen = len(m.runResult.OutputDiff)
			if m.runResult.Stderr != "" {
				outputLen++
				outputLen++
				outputLen += len(components.WrapLines(m.runResult.Stderr, contentWidth))
			}
		} else {
			allOutput := m.runResult.Stdout
			if m.runResult.Stderr != "" {
				if allOutput != "" {
					allOutput += "\nstderr:\n" + m.runResult.Stderr
				} else {
					allOutput = "stderr:\n" + m.runResult.Stderr
				}
			}
			outputLen = len(components.WrapLines(allOutput, contentWidth))
		}
		maxScroll := outputLen - 15
		if maxScroll < 0 {
			maxScroll = 0
		}

		switch msg.String() {
		case "up", "k":
			if m.outputScroll > 0 {
				m.outputScroll--
			}
			return m, nil
		case "down", "j":
			if m.outputScroll < maxScroll {
				m.outputScroll++
			}
			return m, nil
		case "esc", "enter":
			m.selectedProcessIdx = -1
			m.outputScroll = 0
			m.runInputFocused = outputBoxIdx
			return m, nil
		}
		return m, nil
	}

	switch msg.String() {
	case "tab":
		m.detailsTab = 0
		m.detailScroll = 0
		m.runArgsInput.Blur()
		m.runStdinInput.Blur()
		return m, nil

	case "shift+tab":
		m.detailsTab = 2
		m.detailScroll = 0
		m.runArgsInput.Blur()
		m.runStdinInput.Blur()
		return m, nil

	case "down", "j":
		maxIdx := maxFocus
		if m.runResult != nil {
			maxIdx = outputBoxIdx
		}
		if m.runInputFocused < maxIdx {
			m.runInputFocused++
		}
		m.runArgsInput.Blur()
		m.runStdinInput.Blur()
		if m.runInputFocused == 0 {
			m.runArgsInput.Focus()
		} else if m.runInputFocused == 1 {
			m.runStdinInput.Focus()
		}
		return m, nil

	case "up", "k":
		if m.runInputFocused > 0 {
			m.runInputFocused--
		}
		m.runArgsInput.Blur()
		m.runStdinInput.Blur()
		if m.runInputFocused == 0 {
			m.runArgsInput.Focus()
		} else if m.runInputFocused == 1 {
			m.runStdinInput.Focus()
		}
		return m, nil

	case "enter":
		if m.runResult != nil && m.runInputFocused == outputBoxIdx {
			m.selectedProcessIdx = 0
			m.outputScroll = 0
			return m, nil
		}
		if m.runInputFocused == 2 {
			return m, m.executeSubmission()
		} else if m.runInputFocused > 2 && m.runInputFocused <= maxFocus {
			testIdx := m.runInputFocused - 3
			return m, m.executeTestCase(testIdx)
		}

	case "r":
		if m.runInputFocused >= 2 {
			return m, m.executeSubmission()
		}

	case "esc", "q":
		m.currentView = ViewSubmissions
		m.expandedFuncs = nil
		m.clearRunResults()
		m.runArgsInput.Blur()
		m.runStdinInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	if m.runInputFocused == 0 {
		m.runArgsInput, cmd = m.runArgsInput.Update(msg)
	} else if m.runInputFocused == 1 {
		m.runStdinInput, cmd = m.runStdinInput.Update(msg)
	}

	return m, cmd
}

func (m Model) updateExport(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.exportCursor < 1 { // Only 2 options: JSON (0) and CSV (1)
			m.exportCursor++
		}
	case "k", "up":
		if m.exportCursor > 0 {
			m.exportCursor--
		}
	case "enter":
		if m.report != nil {
			return m, m.doExport()
		}
	case "q", "esc":
		m.currentView = ViewSubmissions
	}
	return m, nil
}

func (m Model) updateBannedEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.bannedEditing {
		switch msg.String() {
		case "enter":
			newVal := strings.TrimSpace(m.bannedInput.Value())
			if newVal != "" && m.bannedCursorEdit < len(m.bannedList) {
				m.bannedList[m.bannedCursorEdit] = newVal
			}
			m.bannedEditing = false
			m.bannedInput.Blur()
			return m, nil
		case "esc":
			m.bannedEditing = false
			m.bannedInput.Blur()
			return m, nil
		default:
			var cmd tea.Cmd
			m.bannedInput, cmd = m.bannedInput.Update(msg)
			return m, cmd
		}
	}

	switch msg.String() {
	case "j", "down":
		if len(m.bannedList) > 0 && m.bannedCursorEdit < len(m.bannedList)-1 {
			m.bannedCursorEdit++
		}
	case "k", "up":
		if m.bannedCursorEdit > 0 {
			m.bannedCursorEdit--
		}
	case "enter", "e":
		if len(m.bannedList) > 0 && m.bannedCursorEdit < len(m.bannedList) {
			m.bannedEditing = true
			m.bannedInput.SetValue(m.bannedList[m.bannedCursorEdit])
			m.bannedInput.Focus()
			return m, textinput.Blink
		}
	case "a":
		m.bannedList = append(m.bannedList, "new_function")
		m.bannedCursorEdit = len(m.bannedList) - 1
		m.bannedEditing = true
		m.bannedInput.SetValue("new_function")
		m.bannedInput.Focus()
		return m, textinput.Blink
	case "d", "backspace":
		if len(m.bannedList) > 0 && m.bannedCursorEdit < len(m.bannedList) {
			m.bannedList = append(m.bannedList[:m.bannedCursorEdit], m.bannedList[m.bannedCursorEdit+1:]...)
			if m.bannedCursorEdit >= len(m.bannedList) && m.bannedCursorEdit > 0 {
				m.bannedCursorEdit--
			}
		}
	case "s", "ctrl+s":
		return m, m.saveBannedList()
	case "q", "esc":
		m.currentView = ViewPolicyManage
		return m, m.saveBannedList()
	}
	return m, nil
}

func (m *Model) loadPolicies() tea.Cmd {
	return func() tea.Msg {
		if err := config.Init(); err != nil {
			return errorMsg(err)
		}

		policiesDir, err := config.PoliciesDir()
		if err != nil {
			return errorMsg(err)
		}

		policies, err := policy.Discover(policiesDir)
		if err != nil {
			return errorMsg(err)
		}
		return policiesLoadedMsg(policies)
	}
}

func (m Model) startRun() (tea.Model, tea.Cmd) {
	if len(m.policies) == 0 {
		return m, nil
	}

	selectedPolicy := m.policies[m.selectedPolicy]
	m.currentView = ViewSubmissions
	m.isRunning = true
	m.completed = 0
	m.results = nil
	m.cursor = 0
	m.scrollOffset = 0
	m.runError = ""
	m.executor = nil
	m.clearRunResults()
	m.runID++
	m.submissionsTab = 0
	m.similarityPairsByProcess = make(map[string][]SimilarityPair)
	m.similarityStateByProcess = make(map[string]SimilarityComputeState)
	m.similarityErrorByProcess = make(map[string]string)
	m.similarityProcessNames = nil
	m.similaritySelectedProc = 0
	m.similarityCursor = 0
	m.similarityScroll = 0

	m.resetPairDetailState()

	root := m.root
	keepBinaries := m.settings.KeepBinaries
	shortNames := m.settings.ShortNames

	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			var opts []engine.CompileOption

			if keepBinaries {
				cwd, err := os.Getwd()
				if err == nil {
					binDir := filepath.Join(cwd, "autoscan_binaries")
					opts = append(opts, engine.WithOutputDir(binDir))
				}
			}

			opts = append(opts, engine.WithShortNames(shortNames))

			if m.settings.MaxWorkers > 0 {
				opts = append(opts, engine.WithWorkers(m.settings.MaxWorkers))
			}

			runner, err := engine.NewRunner(selectedPolicy, opts...)
			if err != nil {
				return errorMsg(err)
			}

			if !keepBinaries {
				defer runner.Cleanup()
			}

			report, err := runner.Run(context.Background(), root, engine.RunnerCallbacks{})
			if err != nil {
				return errorMsg(err)
			}

			return runCompleteMsg(*report)
		},
	)
}

func (m Model) doUninstall() tea.Cmd {
	return func() tea.Msg {
		configDir, _ := config.Dir()
		os.RemoveAll(configDir)

		home, _ := os.UserHomeDir()
		os.Remove(filepath.Join(home, ".local", "bin", "autoscan"))

		return uninstallDoneMsg{}
	}
}

func (m Model) loadBannedList() tea.Cmd {
	return func() tea.Msg {
		bannedFile, _ := config.BannedFile()
		funcs, _ := policy.LoadGlobalBanned(bannedFile)
		return bannedListLoadedMsg(funcs)
	}
}

func (m Model) saveBannedList() tea.Cmd {
	return func() tea.Msg {
		bannedFile, _ := config.BannedFile()

		var b strings.Builder
		b.WriteString("# Global Banned Functions\n")
		b.WriteString("banned:\n")
		for _, fn := range m.bannedList {
			b.WriteString(fmt.Sprintf("  - %s\n", fn))
		}

		os.WriteFile(bannedFile, []byte(b.String()), 0644)
		return bannedListSavedMsg{}
	}
}

func (m *Model) initSimilarityProcesses() {
	m.similarityProcessNames = nil
	m.similaritySelectedProc = 0

	if m.selectedPolicy < 0 || m.selectedPolicy >= len(m.policies) {
		return
	}

	pol := m.policies[m.selectedPolicy]
	mp := pol.Run.MultiProcess
	if mp != nil && mp.Enabled && len(mp.Executables) > 0 {
		for _, proc := range mp.Executables {
			m.similarityProcessNames = append(m.similarityProcessNames, proc.Name)
		}
	} else {
		name := pol.Compile.SourceFile
		if name == "" {
			name = "main"
		}
		m.similarityProcessNames = []string{name}
	}

	if m.similarityStateByProcess == nil {
		m.similarityStateByProcess = make(map[string]SimilarityComputeState)
	}
	if m.similarityErrorByProcess == nil {
		m.similarityErrorByProcess = make(map[string]string)
	}
	if m.similarityInFlight == nil {
		m.similarityInFlight = make(map[string]bool)
	}
	for _, p := range m.similarityProcessNames {
		m.similarityStateByProcess[p] = SimilarityNotStarted
	}
}

func (m Model) currentSimilarityProcessName() string {
	if len(m.similarityProcessNames) == 0 {
		return ""
	}
	if m.similaritySelectedProc < 0 || m.similaritySelectedProc >= len(m.similarityProcessNames) {
		return m.similarityProcessNames[0]
	}
	return m.similarityProcessNames[m.similaritySelectedProc]
}

func (m Model) resolveSimilaritySourceFile(process string) string {
	if m.selectedPolicy < 0 || m.selectedPolicy >= len(m.policies) {
		return ""
	}

	pol := m.policies[m.selectedPolicy]
	if mp := pol.Run.MultiProcess; mp != nil && mp.Enabled && len(mp.Executables) > 0 {
		for _, proc := range mp.Executables {
			if proc.Name == process {
				return proc.SourceFile
			}
		}
		return ""
	}
	return pol.Compile.SourceFile
}

func (m Model) computeSimilarityForProcess(process string) tea.Cmd {
	if m.report == nil || len(m.results) == 0 {
		return nil
	}
	if m.similarityPairsByProcess == nil {
		m.similarityPairsByProcess = make(map[string][]SimilarityPair)
	}
	if m.similarityStateByProcess == nil {
		m.similarityStateByProcess = make(map[string]SimilarityComputeState)
	}
	if m.similarityErrorByProcess == nil {
		m.similarityErrorByProcess = make(map[string]string)
	}
	if m.similarityInFlight == nil {
		m.similarityInFlight = make(map[string]bool)
	}
	if m.similarityInFlight[process] {
		return nil
	}
	if m.similarityStateByProcess[process] == SimilarityComputing || m.similarityStateByProcess[process] == SimilarityDone {
		return nil
	}

	srcFile := m.resolveSimilaritySourceFile(process)
	if srcFile == "" {
		if len(m.results) > 0 && len(m.results[0].Submission.CFiles) > 0 {
			srcFile = m.results[0].Submission.CFiles[0]
		}
	}
	if srcFile == "" {
		currentRunID := m.runID
		return func() tea.Msg {
			return similarityErrorMsg{
				process: process,
				runID:   currentRunID,
				err:     fmt.Errorf("no source file found for process %q. check policy configuration", process),
			}
		}
	}

	cfg := domain.CompareConfig{
		WindowSize:     m.settings.PlagiarismWindowSize,
		MinFuncTokens:  m.settings.PlagiarismMinFuncTokens,
		ScoreThreshold: m.settings.PlagiarismScoreThreshold,
	}

	submissions := make([]domain.Submission, len(m.results))
	for i, res := range m.results {
		submissions[i] = res.Submission
	}

	currentRunID := m.runID
	m.similarityInFlight[process] = true
	return tea.Batch(
		func() tea.Msg { return similarityStartedMsg{process: process, runID: currentRunID} },
		func() tea.Msg {
			pairs, err := engine.ComputeSimilarityForProcess(submissions, srcFile, cfg)
			if err != nil {
				return similarityErrorMsg{process: process, err: err, runID: currentRunID}
			}
			return similarityComputedMsg{process: process, pairs: pairs, runID: currentRunID}
		},
	)
}

func (m Model) filteredResults() []domain.SubmissionResult {
	if m.results == nil {
		return nil
	}

	var filtered []domain.SubmissionResult
	query := strings.ToLower(strings.TrimSpace(m.searchQuery))
	for _, r := range m.results {
		switch m.filter {
		case FilterFailed:
			if !r.Compile.OK {
				if query == "" || strings.Contains(strings.ToLower(r.Submission.ID), query) {
					filtered = append(filtered, r)
				}
			}
		case FilterBanned:
			if r.Scan.TotalHits() > 0 {
				if query == "" || strings.Contains(strings.ToLower(r.Submission.ID), query) {
					filtered = append(filtered, r)
				}
			}
		case FilterClean:
			if r.Status == domain.StatusClean {
				if query == "" || strings.Contains(strings.ToLower(r.Submission.ID), query) {
					filtered = append(filtered, r)
				}
			}
		default:
			if query == "" || strings.Contains(strings.ToLower(r.Submission.ID), query) {
				filtered = append(filtered, r)
			}
		}
	}

	return filtered
}

func (m *Model) getExecutor() *engine.Executor {
	if m.executor != nil {
		return m.executor
	}

	if m.selectedPolicy < 0 || m.selectedPolicy >= len(m.policies) {
		return nil
	}

	binDir := ""
	if m.settings.KeepBinaries {
		cwd, err := os.Getwd()
		if err == nil {
			binDir = filepath.Join(cwd, "autoscan_binaries")
		}
	}

	if binDir == "" {
		return nil
	}

	m.executor = engine.NewExecutorWithOptions(m.policies[m.selectedPolicy], binDir, m.settings.ShortNames)
	return m.executor
}

func (m *Model) executeSubmission() tea.Cmd {
	filtered := m.filteredResults()
	if m.cursor >= len(filtered) {
		return nil
	}

	executor := m.getExecutor()
	if executor == nil {
		return func() tea.Msg {
			return executeResultMsg{result: domain.ExecuteResult{
				OK:     false,
				Stderr: "Binaries not available. Enable 'Keep Binaries' in settings and re-run.",
			}}
		}
	}

	sub := filtered[m.cursor].Submission
	argsStr := m.runArgsInput.Value()
	stdinStr := m.runStdinInput.Value()

	var args []string
	if argsStr != "" {
		args = strings.Fields(argsStr)
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.runCancelFunc = cancel
	m.isExecuting = true
	m.selectedProcessIdx = -1
	m.outputScroll = 0

	return func() tea.Msg {
		result := executor.Execute(ctx, sub, args, stdinStr)
		return executeResultMsg{result: result}
	}
}

func (m Model) executeTestCase(testIdx int) tea.Cmd {
	filtered := m.filteredResults()
	if m.cursor >= len(filtered) {
		return nil
	}

	if m.selectedPolicy < 0 || m.selectedPolicy >= len(m.policies) {
		return nil
	}

	pol := m.policies[m.selectedPolicy]
	if testIdx < 0 || testIdx >= len(pol.Run.TestCases) {
		return nil
	}

	executor := m.getExecutor()
	if executor == nil {
		return func() tea.Msg {
			return executeResultMsg{result: domain.ExecuteResult{
				OK:     false,
				Stderr: "Binaries not available. Enable 'Keep Binaries' in settings and re-run.",
			}}
		}
	}

	sub := filtered[m.cursor].Submission
	tc := pol.Run.TestCases[testIdx]

	m.isExecuting = true

	return func() tea.Msg {
		result := executor.ExecuteTestCase(context.Background(), sub, tc)
		return executeResultMsg{result: result}
	}
}

func (m *Model) executeMultiProcess() tea.Cmd {
	filtered := m.filteredResults()
	if m.cursor >= len(filtered) {
		return nil
	}

	if m.selectedPolicy < 0 || m.selectedPolicy >= len(m.policies) {
		return nil
	}

	m.multiProcessResult = nil
	m.showMultiProcess = true

	m.executor = nil
	executor := m.getExecutor()
	if executor == nil || !executor.HasMultiProcess() {
		return func() tea.Msg { return multiProcessResultMsg{result: nil} }
	}

	sub := filtered[m.cursor].Submission
	ctx, cancel := context.WithCancel(context.Background())
	m.runCancelFunc = cancel
	m.isExecuting = true
	m.selectedProcessIdx = -1
	m.outputScroll = 0

	updateChan := make(chan *domain.MultiProcessResult, 100)
	m.multiProcessUpdateChan = updateChan

	go func() {
		result := executor.ExecuteMultiProcess(ctx, sub, func(r *domain.MultiProcessResult) {
			copyResult := copyMultiProcessResult(r)
			select {
			case updateChan <- copyResult:
			default:
			}
		})
		if result != nil {
			select {
			case updateChan <- result:
			default:
			}
		}
		close(updateChan)
	}()

	return waitForMultiProcessUpdates(updateChan)
}

func (m *Model) executeMultiProcessScenario(scenarioIdx int) tea.Cmd {
	filtered := m.filteredResults()
	if m.cursor >= len(filtered) {
		return nil
	}

	if m.selectedPolicy < 0 || m.selectedPolicy >= len(m.policies) {
		return nil
	}

	mp := m.policies[m.selectedPolicy].Run.MultiProcess
	if mp == nil || scenarioIdx < 0 || scenarioIdx >= len(mp.TestScenarios) {
		return nil
	}

	m.multiProcessResult = nil
	m.showMultiProcess = true

	m.executor = nil
	executor := m.getExecutor()
	if executor == nil || !executor.HasMultiProcess() {
		return func() tea.Msg { return multiProcessResultMsg{result: nil} }
	}

	sub := filtered[m.cursor].Submission
	scenario := mp.TestScenarios[scenarioIdx]
	ctx, cancel := context.WithCancel(context.Background())
	m.runCancelFunc = cancel
	m.isExecuting = true
	m.selectedProcessIdx = -1
	m.outputScroll = 0

	updateChan := make(chan *domain.MultiProcessResult, 100)
	m.multiProcessUpdateChan = updateChan

	go func() {
		result := executor.ExecuteMultiProcessScenario(ctx, sub, scenario, func(r *domain.MultiProcessResult) {
			copyResult := copyMultiProcessResult(r)
			select {
			case updateChan <- copyResult:
			default:
			}
		})
		if result != nil {
			select {
			case updateChan <- result:
			default:
			}
		}
		close(updateChan)
	}()

	return waitForMultiProcessUpdates(updateChan)
}

func waitForMultiProcessUpdates(updateChan <-chan *domain.MultiProcessResult) tea.Cmd {
	return func() tea.Msg {
		result, ok := <-updateChan
		if !ok {
			return multiProcessResultMsg{result: nil}
		}
		if result.AllCompleted {
			return multiProcessResultMsg{result: result}
		}
		return multiProcessUpdateMsg{result: result}
	}
}

func copyMultiProcessResult(r *domain.MultiProcessResult) *domain.MultiProcessResult {
	if r == nil {
		return nil
	}
	copy := &domain.MultiProcessResult{
		Processes:     make(map[string]*domain.ProcessResult),
		Order:         append([]string{}, r.Order...),
		TotalDuration: r.TotalDuration,
		AllCompleted:  r.AllCompleted,
		AllPassed:     r.AllPassed,
		ScenarioName:  r.ScenarioName,
	}
	for name, proc := range r.Processes {
		procCopy := *proc
		copy.Processes[name] = &procCopy
	}
	return copy
}
