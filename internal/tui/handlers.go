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
	"github.com/felipetrejos/autoscan/internal/config"
	"github.com/felipetrejos/autoscan/internal/domain"
	"github.com/felipetrejos/autoscan/internal/engine"
	"github.com/felipetrejos/autoscan/internal/policy"
	"github.com/felipetrejos/autoscan/internal/tui/components"
)

// ─────────────────────────────────────────────────────────────────────────────
// Main Update Handler
// ─────────────────────────────────────────────────────────────────────────────

// Update handles all messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// View-specific handling
		switch m.currentView {
		case ViewHome:
			return m.updateHome(msg)
		case ViewPolicySelect:
			return m.updatePolicySelect(msg)
		case ViewPolicyManage:
			return m.updatePolicyManage(msg)
		case ViewPolicyEditor:
			return m.updatePolicyEditor(msg)
		case ViewBannedEditor:
			return m.updateBannedEditor(msg)
		case ViewSettings:
			return m.updateSettings(msg)
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
		// Update help panel width
		m.helpPanel.SetWidth(min(28, m.width/4))

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

	case errorMsg:
		m.runError = msg.Error()
		m.isRunning = false

	case exportDoneMsg:
		m.statusMsg = fmt.Sprintf("Exported to %s", msg.path)

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
	}

	// Update spinner
	var spinnerCmd tea.Cmd
	m.spinner, spinnerCmd = m.spinner.Update(msg)
	cmds = append(cmds, spinnerCmd)

	// Update policy editor if in that view
	if m.currentView == ViewPolicyEditor {
		cmd := m.policyEditor.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// ─────────────────────────────────────────────────────────────────────────────
// Home View Handler
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) updateHome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.menuItem < MenuQuit {
			m.menuItem++
		}
	case "k", "up":
		if m.menuItem > MenuRunGrader {
			m.menuItem--
		}
	case "enter":
		switch m.menuItem {
		case MenuRunGrader:
			m.currentView = ViewPolicySelect
		case MenuManagePolicies:
			m.currentView = ViewPolicyManage
			m.policyManageCursor = 0
		case MenuSettings:
			m.currentView = ViewSettings
			m.settingsCursor = 0
		case MenuUninstall:
			m.confirmDelete = true
		case MenuQuit:
			return m, tea.Quit
		}
	case "y":
		if m.confirmDelete && m.menuItem == MenuUninstall {
			return m, m.doUninstall()
		}
	case "n", "esc":
		m.confirmDelete = false
	case "q":
		if !m.confirmDelete {
			return m, tea.Quit
		}
		m.confirmDelete = false
	case "1":
		m.currentView = ViewPolicySelect
	case "2":
		m.currentView = ViewPolicyManage
		m.policyManageCursor = 0
	case "3":
		m.currentView = ViewSettings
		m.settingsCursor = 0
	case "4":
		m.confirmDelete = true
		m.menuItem = MenuUninstall
	}
	return m, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Policy Select Handler
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) updatePolicySelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.selectedPolicy < len(m.policies)-1 {
			m.selectedPolicy++
		}
	case "k", "up":
		if m.selectedPolicy > 0 {
			m.selectedPolicy--
		}
	case "enter":
		if len(m.policies) > 0 {
			m.currentView = ViewDirectoryInput
			m.folderBrowser.Reset(m.root)
			return m, nil
		}
	case "q", "esc":
		m.currentView = ViewHome
	}
	return m, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Policy Manage Handler
// ─────────────────────────────────────────────────────────────────────────────

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
			m.policyEditor.Reset()
			m.currentView = ViewPolicyEditor
			return m, textinput.Blink
		} else {
			m.policyEditor.Reset()
			m.policyEditor.LoadPolicy(m.policies[m.policyManageCursor-1])
			m.currentView = ViewPolicyEditor
			return m, textinput.Blink
		}
	case "e":
		if m.policyManageCursor > 0 && m.policyManageCursor <= len(m.policies) {
			m.policyEditor.Reset()
			m.policyEditor.LoadPolicy(m.policies[m.policyManageCursor-1])
			m.currentView = ViewPolicyEditor
			return m, textinput.Blink
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

// ─────────────────────────────────────────────────────────────────────────────
// Policy Editor Handler
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) updatePolicyEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.policyEditor.focusedField == FieldCancel || msg.String() == "esc" {
			m.currentView = ViewPolicyManage
			m.policyEditor.errorMsg = ""
			return m, nil
		}
	}

	cmd := m.policyEditor.Update(msg)
	return m, cmd
}

// ─────────────────────────────────────────────────────────────────────────────
// Settings Handler (Updated with KeepBinaries)
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.settingsCursor < 1 {
			m.settingsCursor++
		}
	case "k", "up":
		if m.settingsCursor > 0 {
			m.settingsCursor--
		}
	case "enter", " ":
		// Toggle current setting
		switch m.settingsCursor {
		case 0:
			m.settings.ShortNames = !m.settings.ShortNames
		case 1:
			m.settings.KeepBinaries = !m.settings.KeepBinaries
		}
		config.SaveSettings(m.settings)
	case "q", "esc":
		m.currentView = ViewHome
	}
	return m, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Directory Input Handler
// ─────────────────────────────────────────────────────────────────────────────

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

// ─────────────────────────────────────────────────────────────────────────────
// Submissions Handler
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) updateSubmissions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.isRunning {
		return m, nil
	}

	filtered := m.filteredResults()

	switch msg.String() {
	case "j", "down":
		if m.cursor < len(filtered)-1 {
			m.cursor++
			if m.cursor >= m.scrollOffset+m.visibleRows {
				m.scrollOffset++
			}
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.scrollOffset {
				m.scrollOffset--
			}
		}
	case "enter":
		if len(filtered) > 0 {
			m.currentView = ViewDetails
			m.detailsTab = 0
			m.detailScroll = 0
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
	case "q", "esc":
		m.currentView = ViewHome
		m.results = nil
		m.report = nil
	}

	return m, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Details Handler
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) updateDetails(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Helper to get the number of banned functions for current submission
	getBannedFuncCount := func() int {
		filtered := m.filteredResults()
		if m.cursor >= len(filtered) {
			return 0
		}
		return len(filtered[m.cursor].Scan.HitsByFunction)
	}

	switch msg.String() {
	case "tab":
		m.detailsTab = (m.detailsTab + 1) % 3
		m.detailScroll = 0
		m.bannedCursor = 0
	case "shift+tab":
		m.detailsTab = (m.detailsTab + 2) % 3
		m.detailScroll = 0
		m.bannedCursor = 0
	case "j", "down":
		if m.detailsTab == 1 {
			// Only allow scrolling if there are more functions below
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
				sort.Strings(funcNames) // Must sort - same as view
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
	}
	return m, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Export Handler
// ─────────────────────────────────────────────────────────────────────────────

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

// ─────────────────────────────────────────────────────────────────────────────
// Banned Editor Handler
// ─────────────────────────────────────────────────────────────────────────────

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

// ─────────────────────────────────────────────────────────────────────────────
// Commands
// ─────────────────────────────────────────────────────────────────────────────

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

	root := m.root
	keepBinaries := m.settings.KeepBinaries

	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			var opts []engine.CompileOption

			// If KeepBinaries is enabled, save to CWD/autoscan_binaries
			if keepBinaries {
				cwd, err := os.Getwd()
				if err == nil {
					binDir := filepath.Join(cwd, "autoscan_binaries")
					opts = append(opts, engine.WithOutputDir(binDir))
				}
			}

			runner, err := engine.NewRunner(selectedPolicy, opts...)
			if err != nil {
				return errorMsg(err)
			}

			// Only cleanup binaries if KeepBinaries is false
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

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func (m Model) filteredResults() []domain.SubmissionResult {
	if m.results == nil {
		return nil
	}

	var filtered []domain.SubmissionResult
	for _, r := range m.results {
		switch m.filter {
		case FilterFailed:
			if !r.Compile.OK {
				filtered = append(filtered, r)
			}
		case FilterBanned:
			if r.Scan.TotalHits() > 0 {
				filtered = append(filtered, r)
			}
		case FilterClean:
			if r.Status == domain.StatusClean {
				filtered = append(filtered, r)
			}
		default:
			filtered = append(filtered, r)
		}
	}

	if m.searchQuery != "" {
		var searched []domain.SubmissionResult
		for _, r := range filtered {
			if strings.Contains(strings.ToLower(r.Submission.ID), strings.ToLower(m.searchQuery)) {
				searched = append(searched, r)
			}
		}
		filtered = searched
	}

	return filtered
}
