package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	aipkg "github.com/autoscan-lab/autoscan-engine/pkg/ai"
	"github.com/autoscan-lab/autoscan-engine/pkg/domain"
	"github.com/autoscan-lab/autoscan-engine/pkg/engine"
	"github.com/autoscan-lab/autoscan-engine/pkg/policy"
	"github.com/autoscan-lab/autoscan/internal/config"
	"github.com/autoscan-lab/autoscan/internal/tui/components"
	"github.com/autoscan-lab/autoscan/internal/tui/views/banned"
	"github.com/autoscan-lab/autoscan/internal/tui/views/details"
	"github.com/autoscan-lab/autoscan/internal/tui/views/directory"
	"github.com/autoscan-lab/autoscan/internal/tui/views/home"
	policyview "github.com/autoscan-lab/autoscan/internal/tui/views/policy"
	"github.com/autoscan-lab/autoscan/internal/tui/views/settings"
	"github.com/autoscan-lab/autoscan/internal/tui/views/submissions"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
			result := banned.Update(banned.State{
				Width:            m.width,
				BannedList:       m.bannedList,
				BannedCursorEdit: m.bannedCursorEdit,
				BannedEditing:    m.bannedEditing,
				BannedInput:      m.bannedInput,
			}, msg)
			m.bannedList = result.BannedList
			m.bannedCursorEdit = result.BannedCursorEdit
			m.bannedEditing = result.BannedEditing
			m.bannedInput = result.BannedInput
			if result.Save {
				cmds = append(cmds, m.saveBannedList())
			}
			if result.GoBack {
				m.currentView = ViewPolicyManage
			}
			if result.NeedsInputCmd {
				cmds = append(cmds, textinput.Blink)
			}
			return m, tea.Batch(cmds...)
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
			result := directory.Update(directory.State{
				Width:         m.width,
				InputError:    m.inputError,
				FolderBrowser: m.folderBrowser,
			}, msg)
			m.folderBrowser = result.FolderBrowser
			if result.GoBack {
				m.currentView = ViewPolicySelect
				m.inputError = ""
				return m, nil
			}
			if result.Selected {
				m.root = result.SelectedPath
				m.inputError = ""
				return m.startRun()
			}
			return m, result.Cmd
		case ViewSubmissions:
			result := submissions.Update(m.buildSubmissionsState(), msg)
			m.applySubmissionsResult(result)
			switch result.Nav {
			case submissions.NavGoHome:
				m.currentView = ViewHome
				m.results = nil
				m.report = nil
			case submissions.NavGoDetails:
				m.currentView = ViewDetails
				m.detailsTab = 0
				m.detailScroll = 0
				m.clearRunResults()
				m.executor = nil
			case submissions.NavStartRun:
				return m.startRun()
			}
			if result.ComputeSimilarityFor != "" {
				return m, m.computeSimilarityForProcess(result.ComputeSimilarityFor)
			}
			if result.ComputeAIDetectionFor != "" {
				return m, m.computeAIDetectionForProcess(result.ComputeAIDetectionFor)
			}
			return m, result.Cmd
		case ViewDetails:
			state := m.buildDetailsState()
			result := details.Update(state, msg)
			return m.applyDetailsResult(result)
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
		m.aiReportsByProcess = make(map[string]AIDetectionReport)
		m.aiStateByProcess = make(map[string]SimilarityComputeState)
		m.aiErrorByProcess = make(map[string]string)
		m.aiInFlight = make(map[string]bool)
		m.aiCursor = 0
		m.aiScroll = 0

		m.similaritySelectedProc = 0
		m.aiSelectedProc = 0
		if m.selectedPolicy >= 0 && m.selectedPolicy < len(m.policies) {
			m.similarityProcessNames = submissions.InitSimilarityProcesses(m.policies[m.selectedPolicy])
			m.aiProcessNames = submissions.InitSimilarityProcesses(m.policies[m.selectedPolicy])
		} else {
			m.similarityProcessNames = nil
			m.aiProcessNames = nil
		}
		for _, p := range m.similarityProcessNames {
			m.similarityStateByProcess[p] = SimilarityNotStarted
		}
		for _, p := range m.aiProcessNames {
			m.aiStateByProcess[p] = SimilarityNotStarted
		}

		m.resetPairDetailState()
		m.resetAIDetailState()

	case errorMsg:
		m.runError = msg.Error()
		m.isRunning = false

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

	case aiDetectionStartedMsg:
		if msg.runID == m.runID {
			m.aiStateByProcess[msg.process] = SimilarityComputing
			delete(m.aiErrorByProcess, msg.process)
		}

	case aiDetectionComputedMsg:
		if msg.runID == m.runID {
			m.aiReportsByProcess[msg.process] = msg.report
			m.aiStateByProcess[msg.process] = SimilarityDone
			delete(m.aiErrorByProcess, msg.process)
			delete(m.aiInFlight, msg.process)
		}

	case aiDetectionErrorMsg:
		if msg.runID == m.runID {
			m.aiStateByProcess[msg.process] = SimilarityError
			m.aiErrorByProcess[msg.process] = msg.err.Error()
			delete(m.aiInFlight, msg.process)
		}

	case submissions.PairDetailLoadedMsg:
		if msg.RunID == m.runID && msg.Process == m.pairDetailProcess && msg.PairIndex == m.pairDetailPairIndex {
			m.pairDetailContentA = msg.ContentA
			m.pairDetailContentB = msg.ContentB
			if msg.Err != nil {
				m.pairDetailLoadErr = msg.Err.Error()
			} else {
				m.pairDetailLoadErr = ""
			}
		}

	case submissions.AIDetailLoadedMsg:
		if msg.RunID == m.runID && msg.Process == m.aiDetailProcess && msg.ResultIndex == m.aiDetailResultIndex {
			m.aiDetailContent = msg.Content
			if msg.Err != nil {
				m.aiDetailLoadErr = msg.Err.Error()
			} else {
				m.aiDetailLoadErr = ""
			}
		}

	case policyview.SavedMsg:
		m.currentView = ViewPolicyManage
		m.statusMsg = fmt.Sprintf("Policy saved to %s", msg.Path)
		return m, m.loadPolicies()

	case policyview.SaveErrorMsg:
		m.policyEditor.ErrorMsg = msg.Err

	case policyview.DeletedMsg:
		m.currentView = ViewPolicyManage
		m.statusMsg = fmt.Sprintf("Deleted policy: %s", msg.Name)
		m.confirmDelete = false
		return m, m.loadPolicies()

	case policyview.DeleteErrorMsg:
		m.statusMsg = fmt.Sprintf("Error deleting policy: %s", msg.Err)
		m.confirmDelete = false

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
	oldSelected := m.selectedPolicy
	result := policyview.SelectUpdate(policyview.SelectState{
		Policies:       m.policies,
		SelectedPolicy: m.selectedPolicy,
		InputError:     m.inputError,
	}, msg)

	m.selectedPolicy = result.SelectedPolicy
	m.inputError = result.InputError
	if result.SelectedPolicy != oldSelected {
		m.executor = nil
	}
	if result.GoBack {
		m.currentView = ViewHome
	}
	if result.GoToDirectory {
		m.currentView = ViewDirectoryInput
		m.folderBrowser.Reset(m.root)
	}
	return m, nil
}

func (m Model) buildSubmissionsState() submissions.State {
	policyName := "Unknown Policy"
	if m.selectedPolicy < len(m.policies) {
		policyName = m.policies[m.selectedPolicy].Name
	}

	// Pre-compute source files by process from policies
	sourceFileByProcess := make(map[string]string)
	var currentPolicy *policy.Policy
	if m.selectedPolicy >= 0 && m.selectedPolicy < len(m.policies) {
		currentPolicy = m.policies[m.selectedPolicy]
	}
	for _, proc := range m.similarityProcessNames {
		sourceFileByProcess[proc] = submissions.ResolveSourceFile(currentPolicy, proc)
	}
	for _, proc := range m.aiProcessNames {
		if _, ok := sourceFileByProcess[proc]; !ok {
			sourceFileByProcess[proc] = submissions.ResolveSourceFile(currentPolicy, proc)
		}
	}

	return submissions.State{
		Width:       m.width,
		Height:      m.height,
		VisibleRows: m.visibleRows,

		PolicyName: policyName,
		Root:       m.root,

		Report:    m.report,
		Results:   m.results,
		Filtered:  submissions.FilterResults(m.results, int(m.filter), m.searchQuery),
		IsRunning: m.isRunning,
		RunError:  m.runError,
		RunID:     m.runID,

		Settings: m.settings,

		SubmissionsTab: m.submissionsTab,
		Cursor:         m.cursor,
		ScrollOffset:   m.scrollOffset,
		Filter:         int(m.filter),

		SearchInput:  m.searchInput,
		SearchActive: m.searchActive,
		SearchQuery:  m.searchQuery,

		SimilarityProcessNames:   m.similarityProcessNames,
		SimilaritySelectedProc:   m.similaritySelectedProc,
		SimilarityPairsByProcess: m.similarityPairsByProcess,
		SimilarityStateByProcess: m.convertSimilarityState(),
		SimilarityErrorByProcess: m.similarityErrorByProcess,
		SimilarityInFlight:       m.similarityInFlight,
		SimilarityCursor:         m.similarityCursor,
		SimilarityScroll:         m.similarityScroll,

		AIProcessNames:     m.aiProcessNames,
		AISelectedProc:     m.aiSelectedProc,
		AIReportsByProcess: m.aiReportsByProcess,
		AIStateByProcess:   m.convertAIState(),
		AIErrorByProcess:   m.aiErrorByProcess,
		AIInFlight:         m.aiInFlight,
		AICursor:           m.aiCursor,
		AIScroll:           m.aiScroll,

		PairDetailOpen:        m.pairDetailOpen,
		PairDetailProcess:     m.pairDetailProcess,
		PairDetailPairIndex:   m.pairDetailPairIndex,
		PairDetailContentA:    m.pairDetailContentA,
		PairDetailContentB:    m.pairDetailContentB,
		PairDetailLoadErr:     m.pairDetailLoadErr,
		PairDetailFocusedPane: m.pairDetailFocusedPane,
		PairDetailScrollA:     m.pairDetailScrollA,
		PairDetailScrollB:     m.pairDetailScrollB,
		PairDetailHScrollA:    m.pairDetailHScrollA,
		PairDetailHScrollB:    m.pairDetailHScrollB,

		AIDetailOpen:        m.aiDetailOpen,
		AIDetailProcess:     m.aiDetailProcess,
		AIDetailResultIndex: m.aiDetailResultIndex,
		AIDetailContent:     m.aiDetailContent,
		AIDetailLoadErr:     m.aiDetailLoadErr,
		AIDetailScroll:      m.aiDetailScroll,
		AIDetailHScroll:     m.aiDetailHScroll,

		Spinner:             m.spinner.View(),
		SourceFileByProcess: sourceFileByProcess,
	}
}

func (m Model) convertSimilarityState() map[string]submissions.SimilarityComputeState {
	result := make(map[string]submissions.SimilarityComputeState)
	for k, v := range m.similarityStateByProcess {
		result[k] = submissions.SimilarityComputeState(v)
	}
	return result
}

func (m Model) convertAIState() map[string]submissions.SimilarityComputeState {
	result := make(map[string]submissions.SimilarityComputeState)
	for k, v := range m.aiStateByProcess {
		result[k] = submissions.SimilarityComputeState(v)
	}
	return result
}

func (m *Model) applySubmissionsResult(r submissions.UpdateResult) {
	m.submissionsTab = r.SubmissionsTab
	m.cursor = r.Cursor
	m.scrollOffset = r.ScrollOffset
	m.filter = Filter(r.Filter)

	m.searchInput = r.SearchInput
	m.searchActive = r.SearchActive
	m.searchQuery = r.SearchQuery

	m.similaritySelectedProc = r.SimilaritySelectedProc
	m.similarityCursor = r.SimilarityCursor
	m.similarityScroll = r.SimilarityScroll
	m.aiSelectedProc = r.AISelectedProc
	m.aiCursor = r.AICursor
	m.aiScroll = r.AIScroll

	if r.SimilarityPairsByProcess != nil {
		m.similarityPairsByProcess = r.SimilarityPairsByProcess
	}
	if r.SimilarityStateByProcess != nil {
		for k, v := range r.SimilarityStateByProcess {
			m.similarityStateByProcess[k] = SimilarityComputeState(v)
		}
	}
	if r.SimilarityErrorByProcess != nil {
		m.similarityErrorByProcess = r.SimilarityErrorByProcess
	}
	if r.AIReportsByProcess != nil {
		m.aiReportsByProcess = r.AIReportsByProcess
	}
	if r.AIStateByProcess != nil {
		for k, v := range r.AIStateByProcess {
			m.aiStateByProcess[k] = SimilarityComputeState(v)
		}
	}
	if r.AIErrorByProcess != nil {
		m.aiErrorByProcess = r.AIErrorByProcess
	}

	m.pairDetailOpen = r.PairDetailOpen
	m.pairDetailProcess = r.PairDetailProcess
	m.pairDetailPairIndex = r.PairDetailPairIndex
	m.pairDetailContentA = r.PairDetailContentA
	m.pairDetailContentB = r.PairDetailContentB
	m.pairDetailLoadErr = r.PairDetailLoadErr
	m.pairDetailFocusedPane = r.PairDetailFocusedPane
	m.pairDetailScrollA = r.PairDetailScrollA
	m.pairDetailScrollB = r.PairDetailScrollB
	m.pairDetailHScrollA = r.PairDetailHScrollA
	m.pairDetailHScrollB = r.PairDetailHScrollB
	m.aiDetailOpen = r.AIDetailOpen
	m.aiDetailProcess = r.AIDetailProcess
	m.aiDetailResultIndex = r.AIDetailResultIndex
	m.aiDetailContent = r.AIDetailContent
	m.aiDetailLoadErr = r.AIDetailLoadErr
	m.aiDetailScroll = r.AIDetailScroll
	m.aiDetailHScroll = r.AIDetailHScroll

	if r.ClearResults {
		m.clearRunResults()
	}
}

func (m Model) updatePolicyManage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	result := policyview.ManageUpdate(policyview.ManageState{
		Policies:           m.policies,
		PolicyManageCursor: m.policyManageCursor,
		ConfirmDelete:      m.confirmDelete,
	}, msg)

	m.policyManageCursor = result.PolicyManageCursor
	m.confirmDelete = result.ConfirmDelete

	switch result.Navigation {
	case policyview.ManageNavBack:
		m.currentView = ViewHome
	case policyview.ManageNavBannedEditor:
		return m, m.loadBannedList()
	case policyview.ManageNavNewPolicy:
		return m, m.openPolicyEditor(nil)
	case policyview.ManageNavEditPolicy:
		if result.PolicyToEdit != nil {
			return m, m.openPolicyEditor(result.PolicyToEdit)
		}
	}
	if result.PolicyToDelete != nil {
		return m, policyview.DeletePolicy(result.PolicyToDelete)
	}
	return m, nil
}

func (m Model) updatePolicyEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" && !m.policyEditor.InSubMode() {
		m.currentView = ViewPolicyManage
		m.policyEditor.ErrorMsg = ""
		return m, nil
	}

	cmd := m.policyEditor.Update(msg)
	return m, cmd
}

func (m Model) buildDetailsState() details.State {
	filtered := submissions.FilterResults(m.results, int(m.filter), m.searchQuery)
	var result domain.SubmissionResult
	var submissionID string
	if m.cursor < len(filtered) {
		result = filtered[m.cursor]
		submissionID = result.Submission.ID
	}

	// Determine if multi-process mode
	isMultiProcess := false
	var testCases []policy.TestCase
	var testScenarios []policy.MultiProcessScenario
	var multiProcessExecs []policy.ProcessConfig

	if m.selectedPolicy >= 0 && m.selectedPolicy < len(m.policies) {
		p := m.policies[m.selectedPolicy]
		testCases = p.Run.TestCases
		if mp := p.Run.MultiProcess; mp != nil && mp.Enabled && len(mp.Executables) > 0 {
			isMultiProcess = true
			testScenarios = mp.TestScenarios
			multiProcessExecs = mp.Executables
		}
	}

	return details.State{
		Width:              m.width,
		Height:             m.height,
		Result:             result,
		SubmissionID:       submissionID,
		DetailsTab:         m.detailsTab,
		DetailScroll:       m.detailScroll,
		BannedCursor:       m.bannedCursor,
		ExpandedFuncs:      m.expandedFuncs,
		RunInputFocused:    m.runInputFocused,
		SelectedProcessIdx: m.selectedProcessIdx,
		OutputScroll:       m.outputScroll,
		IsExecuting:        m.isExecuting,
		SpinnerView:        m.spinner.View(),
		RunArgsInput:       m.runArgsInput,
		RunStdinInput:      m.runStdinInput,
		RunResult:          m.runResult,
		RunTestResults:     m.runTestResults,
		MultiProcessResult: m.multiProcessResult,
		ShowMultiProcess:   m.showMultiProcess,
		IsMultiProcess:     isMultiProcess,
		TestCases:          testCases,
		TestScenarios:      testScenarios,
		MultiProcessExecs:  multiProcessExecs,
		KeepBinaries:       m.settings.KeepBinaries,
	}
}

func (m Model) applyDetailsResult(result details.UpdateResult) (tea.Model, tea.Cmd) {
	m.detailsTab = result.DetailsTab
	m.detailScroll = result.DetailScroll
	m.bannedCursor = result.BannedCursor
	m.expandedFuncs = result.ExpandedFuncs
	m.runInputFocused = result.RunInputFocused
	m.selectedProcessIdx = result.SelectedProcessIdx
	m.outputScroll = result.OutputScroll
	m.showMultiProcess = result.ShowMultiProcess
	m.runArgsInput = result.RunArgsInput
	m.runStdinInput = result.RunStdinInput

	if result.GoBack {
		m.currentView = ViewSubmissions
		m.expandedFuncs = nil
		m.bannedCursor = 0
		m.clearRunResults()
		m.runArgsInput.Blur()
		m.runStdinInput.Blur()
		return m, nil
	}

	if result.CancelExecution {
		if m.runCancelFunc != nil {
			m.runCancelFunc()
			m.runCancelFunc = nil
		}
		m.isExecuting = false
		m.statusMsg = "Processes killed (SIGKILL)"
		return m, nil
	}

	if result.ExecuteSubmission {
		return m, m.executeSubmission()
	}

	if result.ExecuteTestCase >= 0 {
		return m, m.executeTestCase(result.ExecuteTestCase)
	}

	if result.ExecuteMultiProcess {
		return m, m.executeMultiProcess()
	}

	if result.ExecuteScenario >= 0 {
		return m, m.executeMultiProcessScenario(result.ExecuteScenario)
	}

	return m, result.Cmd
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
	m.similarityInFlight = make(map[string]bool)
	m.similarityProcessNames = nil
	m.similaritySelectedProc = 0
	m.similarityCursor = 0
	m.similarityScroll = 0
	m.aiReportsByProcess = make(map[string]AIDetectionReport)
	m.aiStateByProcess = make(map[string]SimilarityComputeState)
	m.aiErrorByProcess = make(map[string]string)
	m.aiInFlight = make(map[string]bool)
	m.aiProcessNames = nil
	m.aiSelectedProc = 0
	m.aiCursor = 0
	m.aiScroll = 0

	m.resetPairDetailState()
	m.resetAIDetailState()

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

	var currentPolicy *policy.Policy
	if m.selectedPolicy >= 0 && m.selectedPolicy < len(m.policies) {
		currentPolicy = m.policies[m.selectedPolicy]
	}
	srcFile := submissions.ResolveSourceFile(currentPolicy, process)
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
			report, err := engine.ComputeSimilarityForProcess(submissions, srcFile, cfg)
			if err != nil {
				return similarityErrorMsg{process: process, err: err, runID: currentRunID}
			}
			return similarityComputedMsg{process: process, pairs: report.Pairs, runID: currentRunID}
		},
	)
}

func (m Model) computeAIDetectionForProcess(process string) tea.Cmd {
	if m.report == nil || len(m.results) == 0 {
		return nil
	}
	if m.aiReportsByProcess == nil {
		m.aiReportsByProcess = make(map[string]AIDetectionReport)
	}
	if m.aiStateByProcess == nil {
		m.aiStateByProcess = make(map[string]SimilarityComputeState)
	}
	if m.aiErrorByProcess == nil {
		m.aiErrorByProcess = make(map[string]string)
	}
	if m.aiInFlight == nil {
		m.aiInFlight = make(map[string]bool)
	}
	if m.aiInFlight[process] {
		return nil
	}
	if m.aiStateByProcess[process] == SimilarityComputing || m.aiStateByProcess[process] == SimilarityDone {
		return nil
	}

	var currentPolicy *policy.Policy
	if m.selectedPolicy >= 0 && m.selectedPolicy < len(m.policies) {
		currentPolicy = m.policies[m.selectedPolicy]
	}
	srcFile := submissions.ResolveSourceFile(currentPolicy, process)
	if srcFile == "" {
		if len(m.results) > 0 && len(m.results[0].Submission.CFiles) > 0 {
			srcFile = m.results[0].Submission.CFiles[0]
		}
	}
	if srcFile == "" {
		currentRunID := m.runID
		return func() tea.Msg {
			return aiDetectionErrorMsg{
				process: process,
				runID:   currentRunID,
				err:     fmt.Errorf("no source file found for process %q. check policy configuration", process),
			}
		}
	}

	dictPath, err := config.AIDictionaryFile()
	if err != nil {
		currentRunID := m.runID
		return func() tea.Msg {
			return aiDetectionErrorMsg{
				process: process,
				runID:   currentRunID,
				err:     fmt.Errorf("resolving ai dictionary path: %w", err),
			}
		}
	}
	dict, err := aipkg.LoadDictionary(dictPath)
	if err != nil {
		currentRunID := m.runID
		return func() tea.Msg {
			return aiDetectionErrorMsg{
				process: process,
				runID:   currentRunID,
				err:     fmt.Errorf("loading ai dictionary: %w", err),
			}
		}
	}

	cfg := domain.CompareConfig{
		WindowSize:     m.settings.AIWindowSize,
		MinFuncTokens:  m.settings.AIMinFuncTokens,
		ScoreThreshold: m.settings.AIScoreThreshold,
	}

	subs := make([]domain.Submission, len(m.results))
	for i, res := range m.results {
		subs[i] = res.Submission
	}

	currentRunID := m.runID
	m.aiInFlight[process] = true
	return tea.Batch(
		func() tea.Msg { return aiDetectionStartedMsg{process: process, runID: currentRunID} },
		func() tea.Msg {
			report, err := engine.ComputeAIDetectionForProcess(subs, srcFile, dict, cfg)
			if err != nil {
				return aiDetectionErrorMsg{process: process, err: err, runID: currentRunID}
			}
			return aiDetectionComputedMsg{process: process, report: report, runID: currentRunID}
		},
	)
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
	filtered := submissions.FilterResults(m.results, int(m.filter), m.searchQuery)
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
	filtered := submissions.FilterResults(m.results, int(m.filter), m.searchQuery)
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
	filtered := submissions.FilterResults(m.results, int(m.filter), m.searchQuery)
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
	filtered := submissions.FilterResults(m.results, int(m.filter), m.searchQuery)
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
