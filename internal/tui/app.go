package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/felipetrejos/autoscan/internal/config"
	"github.com/felipetrejos/autoscan/internal/domain"
	"github.com/felipetrejos/autoscan/internal/engine"
	"github.com/felipetrejos/autoscan/internal/export"
	"github.com/felipetrejos/autoscan/internal/policy"
	"github.com/felipetrejos/autoscan/internal/tui/styles"
)

// ASCII Art Logo
const logo = `
  ╔═══════════════════════════════════════════════╗
  ║                                               ║
  ║      █████╗ ██╗   ██╗████████╗ ██████╗        ║
  ║     ██╔══██╗██║   ██║╚══██╔══╝██╔═══██╗       ║
  ║     ███████║██║   ██║   ██║   ██║   ██║       ║
  ║     ██╔══██║██║   ██║   ██║   ██║   ██║       ║
  ║     ██║  ██║╚██████╔╝   ██║   ╚██████╔╝       ║
  ║     ╚═╝  ╚═╝ ╚═════╝    ╚═╝    ╚═════╝        ║
  ║                                               ║
  ║     ███████╗ ██████╗ █████╗ ███╗   ██╗        ║
  ║     ██╔════╝██╔════╝██╔══██╗████╗  ██║        ║
  ║     ███████╗██║     ███████║██╔██╗ ██║        ║
  ║     ╚════██║██║     ██╔══██║██║╚██╗██║        ║
  ║     ███████║╚██████╗██║  ██║██║ ╚████║        ║
  ║     ╚══════╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═══╝        ║
  ║                                               ║
  ║        C Lab Submission Grader                ║
  ║                                               ║
  ╚═══════════════════════════════════════════════╝
`

// View represents different screens in the app
type View int

const (
	ViewHome View = iota
	ViewPolicySelect
	ViewPolicyManage
	ViewPolicyEditor
	ViewDirectoryInput
	ViewSubmissions
	ViewDetails
	ViewHelp
	ViewExport
)

// Filter represents the current filter mode
type Filter int

const (
	FilterAll Filter = iota
	FilterFailed
	FilterBanned
	FilterClean
)

func (f Filter) String() string {
	switch f {
	case FilterFailed:
		return "Compile Fails"
	case FilterBanned:
		return "Banned Calls"
	case FilterClean:
		return "Clean"
	default:
		return "All"
	}
}

// Menu items for home screen
type MenuItem int

const (
	MenuRunGrader MenuItem = iota
	MenuManagePolicies
	MenuHelp
	MenuUninstall
	MenuQuit
)

// Messages for async updates
type (
	policiesLoadedMsg    []*policy.Policy
	discoveryCompleteMsg []domain.Submission
	compileCompleteMsg   struct {
		sub    domain.Submission
		result domain.CompileResult
	}
	runCompleteMsg domain.RunReport
	errorMsg       error
	exportDoneMsg  struct {
		format string
		path   string
	}
)

// Model is the main Bubble Tea model
type Model struct {
	currentView View
	width       int
	height      int

	// Home menu
	menuItem MenuItem

	// Policy selection
	policies       []*policy.Policy
	selectedPolicy int

	// Policy management
	policyManageCursor int
	policyEditor       PolicyEditor
	confirmDelete      bool

	// Directory browser
	folderBrowser FolderBrowser
	root          string
	inputError    string

	// Runner
	runner *engine.Runner

	// Current run state
	submissions []domain.Submission
	results     []domain.SubmissionResult
	report      *domain.RunReport
	isRunning   bool
	completed   int
	spinner     spinner.Model
	runError    string

	// List navigation
	cursor       int
	scrollOffset int
	visibleRows  int
	filter       Filter
	searchQuery  string
	searchActive bool

	// Details view
	detailsTab   int
	detailScroll int

	// Export
	exportCursor int

	// Status message
	statusMsg string
}

// Config holds TUI startup configuration
type Config struct {
	PolicyPath string
	Root       string
}

// New creates a new TUI model
func New(cfg Config) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	root := cfg.Root
	if root == "" {
		root = "."
	}

	return Model{
		currentView:   ViewHome,
		root:          root,
		spinner:       s,
		folderBrowser: NewFolderBrowser(root),
		visibleRows:   20,
		filter:        FilterAll,
		menuItem:      MenuRunGrader,
		policyEditor:  NewPolicyEditor(80, 40),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadPolicies(),
		m.spinner.Tick,
	)
}

func (m *Model) loadPolicies() tea.Cmd {
	return func() tea.Msg {
		// Ensure config directory exists with defaults
		if err := config.Init(); err != nil {
			return errorMsg(err)
		}

		// Load from config directory
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
		case ViewDirectoryInput:
			return m.updateDirectoryInput(msg)
		case ViewSubmissions:
			return m.updateSubmissions(msg)
		case ViewDetails:
			return m.updateDetails(msg)
		case ViewHelp:
			return m.updateHelp(msg)
		case ViewExport:
			return m.updateExport(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.visibleRows = msg.Height - 12
		if m.visibleRows < 5 {
			m.visibleRows = 5
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case policiesLoadedMsg:
		m.policies = msg
		m.statusMsg = ""

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
		// Print message and quit
		fmt.Println("\nautoscan has been uninstalled.")
		fmt.Println("Config removed from ~/.config/autoscan/")
		fmt.Println("Binary removed from /usr/local/bin/autoscan")
		return m, tea.Quit
	}

	// Update policy editor if in that view
	if m.currentView == ViewPolicyEditor {
		cmd := m.policyEditor.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

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
		case MenuHelp:
			m.currentView = ViewHelp
		case MenuUninstall:
			m.confirmDelete = true // Reuse for uninstall confirmation
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
		m.currentView = ViewHelp
	case "4":
		m.confirmDelete = true
		m.menuItem = MenuUninstall
	}
	return m, nil
}

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

func (m Model) updatePolicyManage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Extra item for "Create New" at index 0
	maxCursor := len(m.policies) // 0 is "Create New", 1..len are policies

	switch msg.String() {
	case "j", "down":
		if m.policyManageCursor < maxCursor {
			m.policyManageCursor++
		}
	case "k", "up":
		if m.policyManageCursor > 0 {
			m.policyManageCursor--
		}
	case "enter":
		if m.policyManageCursor == 0 {
			// Create new policy
			m.policyEditor.Reset()
			m.currentView = ViewPolicyEditor
			return m, textinput.Blink
		} else {
			// Edit selected policy (cursor - 1 since 0 is "Create New")
			m.policyEditor.Reset()
			m.policyEditor.LoadPolicy(m.policies[m.policyManageCursor-1])
			m.currentView = ViewPolicyEditor
			return m, textinput.Blink
		}
	case "e":
		// Edit selected policy
		if m.policyManageCursor > 0 && m.policyManageCursor <= len(m.policies) {
			m.policyEditor.Reset()
			m.policyEditor.LoadPolicy(m.policies[m.policyManageCursor-1])
			m.currentView = ViewPolicyEditor
			return m, textinput.Blink
		}
	case "d":
		// Delete selected policy (only for existing policies, not "Create New")
		if m.policyManageCursor > 0 && m.policyManageCursor <= len(m.policies) {
			m.confirmDelete = true
		}
	case "y":
		// Confirm delete
		if m.confirmDelete && m.policyManageCursor > 0 && m.policyManageCursor <= len(m.policies) {
			return m, DeletePolicy(m.policies[m.policyManageCursor-1])
		}
	case "n":
		// Cancel delete
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
	switch msg.String() {
	case "esc":
		if m.policyEditor.focusedField == FieldCancel || msg.String() == "esc" {
			m.currentView = ViewPolicyManage
			m.policyEditor.errorMsg = ""
			return m, nil
		}
	}

	// Let the editor handle other keys
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

	// Let folder browser handle navigation
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

func (m Model) updateDetails(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.detailsTab = (m.detailsTab + 1) % 3
		m.detailScroll = 0
	case "shift+tab":
		m.detailsTab = (m.detailsTab + 2) % 3
		m.detailScroll = 0
	case "j", "down":
		m.detailScroll++
	case "k", "up":
		if m.detailScroll > 0 {
			m.detailScroll--
		}
	case "q", "esc":
		m.currentView = ViewSubmissions
	}
	return m, nil
}

func (m Model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "enter":
		m.currentView = ViewHome
	}
	return m, nil
}

func (m Model) updateExport(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.exportCursor < 2 {
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

func (m Model) doExport() tea.Cmd {
	return func() tea.Msg {
		outputDir := "."
		var path string
		var err error
		var format string

		switch m.exportCursor {
		case 0:
			format = "Markdown"
			path, err = export.Markdown(*m.report, outputDir)
		case 1:
			format = "JSON"
			path, err = export.JSON(*m.report, outputDir)
		case 2:
			format = "CSV"
			path, err = export.CSV(*m.report, outputDir)
		}

		if err != nil {
			return errorMsg(err)
		}
		return exportDoneMsg{format: format, path: path}
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

	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			runner, err := engine.NewRunner(selectedPolicy)
			if err != nil {
				return errorMsg(err)
			}
			defer runner.Cleanup()

			report, err := runner.Run(context.Background(), root, engine.RunnerCallbacks{})
			if err != nil {
				return errorMsg(err)
			}

			return runCompleteMsg(*report)
		},
	)
}

type uninstallDoneMsg struct{}

func (m Model) doUninstall() tea.Cmd {
	return func() tea.Msg {
		// Remove config directory
		configDir, _ := config.Dir()
		os.RemoveAll(configDir)

		// Remove binary from ~/.local/bin
		home, _ := os.UserHomeDir()
		os.Remove(filepath.Join(home, ".local", "bin", "autoscan"))

		return uninstallDoneMsg{}
	}
}

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

func (m Model) View() string {
	switch m.currentView {
	case ViewHome:
		return m.renderHome()
	case ViewPolicySelect:
		return m.renderPolicySelect()
	case ViewPolicyManage:
		return m.renderPolicyManage()
	case ViewPolicyEditor:
		return m.policyEditor.View()
	case ViewDirectoryInput:
		return m.renderDirectoryInput()
	case ViewSubmissions:
		return m.renderSubmissions()
	case ViewDetails:
		return m.renderDetails()
	case ViewHelp:
		return m.renderHelp()
	case ViewExport:
		return m.renderExport()
	default:
		return m.renderHome()
	}
}

func (m Model) renderHome() string {
	var b strings.Builder

	// Logo
	logoStyle := lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true)
	b.WriteString(logoStyle.Render(logo))
	b.WriteString("\n")

	// Menu box
	menuBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(1, 3).
		Width(40)

	var menu strings.Builder
	menuItems := []struct {
		key  string
		desc string
		item MenuItem
	}{
		{"1", "Run Grader", MenuRunGrader},
		{"2", "Manage Policies", MenuManagePolicies},
		{"3", "Help", MenuHelp},
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
		menu.WriteString(styles.WarningStyle.Render("Remove autoscan and all configs?"))
		menu.WriteString("\n")
		menu.WriteString(styles.Subtle.Render("[y] confirm  [n] cancel"))
	}

	b.WriteString(lipgloss.Place(
		m.width,
		3,
		lipgloss.Center,
		lipgloss.Center,
		menuBox.Render(menu.String()),
	))

	b.WriteString("\n\n")
	b.WriteString(lipgloss.Place(
		m.width,
		1,
		lipgloss.Center,
		lipgloss.Center,
		styles.Subtle.Render("Use ↑/↓ to navigate, Enter to select"),
	))

	return b.String()
}

func (m Model) renderPolicySelect() string {
	var b strings.Builder

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		Padding(1, 2)

	b.WriteString(header.Render("Select a Policy"))
	b.WriteString("\n\n")

	if len(m.policies) == 0 {
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Warning).
			Padding(1, 2)

		content := styles.WarningStyle.Render("No policies found!") + "\n\n" +
			styles.Subtle.Render("Create a policy via Manage Policies or edit ~/.config/autoscan/")

		b.WriteString(box.Render(content))
	} else {
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Muted).
			Padding(1, 2).
			Width(60)

		var list strings.Builder
		for i, p := range m.policies {
			cursor := "  "
			style := styles.NormalItem
			if i == m.selectedPolicy {
				cursor = "▸ "
				style = styles.SelectedItem
			}

			list.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(p.Name)))

			if i == m.selectedPolicy {
				relPath, _ := filepath.Rel(".", p.FilePath)
				list.WriteString(styles.Subtle.Render(fmt.Sprintf("    %s\n", relPath)))

				// Show compiler flags
				if len(p.Compile.Flags) > 0 {
					list.WriteString(styles.Subtle.Render(fmt.Sprintf("    flags: %s\n", strings.Join(p.Compile.Flags, " "))))
				}
			}
		}

		b.WriteString(box.Render(list.String()))
	}

	b.WriteString("\n\n")
	b.WriteString(m.renderHelpBar([]helpItem{
		{"↑/↓", "navigate"},
		{"enter", "select"},
		{"esc", "back"},
	}))

	return b.String()
}

func (m Model) renderDirectoryInput() string {
	var b strings.Builder

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		Padding(1, 2)

	b.WriteString(header.Render("Select Submissions Folder"))
	b.WriteString("\n\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(1, 2).
		Width(60)

	b.WriteString(box.Render(m.folderBrowser.View()))

	if m.inputError != "" {
		b.WriteString("\n")
		b.WriteString(styles.ErrorStyle.Render("Error: " + m.inputError))
	}

	b.WriteString("\n\n")
	b.WriteString(m.renderHelpBar([]helpItem{
		{"↑/↓", "navigate"},
		{"enter", "open/select"},
		{"←", "go up"},
		{"esc", "back"},
	}))

	return b.String()
}

func (m Model) renderPolicyManage() string {
	var b strings.Builder

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		Padding(1, 2)

	b.WriteString(header.Render("Manage Policies"))
	b.WriteString("\n\n")

	// Create new option
	createStyle := styles.NormalItem
	cursor := "  "
	if m.policyManageCursor == 0 {
		createStyle = styles.SelectedItem
		cursor = "▸ "
	}
	b.WriteString(fmt.Sprintf("%s%s %s\n\n", cursor, styles.SuccessStyle.Render("+"), createStyle.Render("Create New Policy")))

	// List existing policies
	if len(m.policies) == 0 {
		b.WriteString(styles.Subtle.Render("  No policies found. Create one to get started.\n"))
	} else {
		for i, pol := range m.policies {
			cursor := "  "
			style := styles.NormalItem
			idx := i + 1 // +1 because 0 is "Create New"
			if m.policyManageCursor == idx {
				cursor = "▸ "
				style = styles.SelectedItem
			}

			// Show flags info instead of banned count
			info := ""
			if len(pol.Compile.Flags) > 0 {
				info = styles.Subtle.Render(fmt.Sprintf("[%s]", strings.Join(pol.Compile.Flags, " ")))
			}

			b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, style.Render(pol.Name), info))
		}
	}

	// Show delete confirmation if active
	if m.confirmDelete {
		b.WriteString("\n")
		deleteBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Danger).
			Padding(1, 2).
			Width(50)

		selectedPolicy := m.policies[m.policyManageCursor-1]
		confirmContent := fmt.Sprintf(
			"%s\n\n%s",
			styles.ErrorStyle.Render("Delete policy \""+selectedPolicy.Name+"\"?"),
			styles.Subtle.Render("Press [y] to confirm, [n] to cancel"),
		)
		b.WriteString(deleteBox.Render(confirmContent))
	}

	// Status message
	if m.statusMsg != "" {
		b.WriteString("\n\n")
		b.WriteString(styles.SuccessStyle.Render("✓ " + m.statusMsg))
	}

	b.WriteString("\n\n")
	helpItems := []helpItem{
		{"↑/↓", "navigate"},
		{"enter", "edit"},
		{"d", "delete"},
		{"esc", "back"},
	}
	b.WriteString(m.renderHelpBar(helpItems))

	return b.String()
}

func (m Model) renderPolicyEditor() string {
	return m.policyEditor.View()
}

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
	b.WriteString(styles.Subtle.Render(fmt.Sprintf("  %s", m.root)))
	b.WriteString("\n")

	if m.runError != "" {
		b.WriteString("\n")
		errorBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Error).
			Padding(1, 2)
		b.WriteString(errorBox.Render(styles.ErrorStyle.Render("Error: " + m.runError)))
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

	// Table header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary)

	table.WriteString(headerStyle.Render(fmt.Sprintf("  %-4s  %-30s  %-7s  %s",
		"", "Submission", "Compile", "Info")))
	table.WriteString("\n")
	table.WriteString(strings.Repeat("─", 70))
	table.WriteString("\n")

	// Results list
	filtered := m.filteredResults()

	if len(filtered) == 0 && !m.isRunning {
		table.WriteString(styles.Subtle.Render("  No submissions found"))
		table.WriteString("\n")
	}

	endIdx := m.scrollOffset + m.visibleRows
	if endIdx > len(filtered) {
		endIdx = len(filtered)
	}

	for i := m.scrollOffset; i < endIdx; i++ {
		r := filtered[i]

		// Status icon (missing files adds a warning marker)
		// Status
		var statusIcon, statusStyle string
		switch r.Status {
		case domain.StatusClean:
			if r.Submission.HasMissingFiles() {
				statusIcon, statusStyle = "[~]", "warning"
			} else {
				statusIcon, statusStyle = "[OK]", "success"
			}
		case domain.StatusBanned:
			statusIcon, statusStyle = "[!]", "warning"
		case domain.StatusFailed, domain.StatusTimedOut:
			statusIcon, statusStyle = "[X]", "error"
		default:
			statusIcon, statusStyle = "...", ""
		}

		// Compile status
		var compileStr, compileStyle string
		if r.Compile.TimedOut {
			compileStr, compileStyle = "TIMEOUT", "warning"
		} else if !r.Compile.OK {
			compileStr, compileStyle = "FAIL", "error"
		} else {
			compileStr, compileStyle = "OK", "success"
		}

		// Truncate ID if needed
		id := r.Submission.ID
		if len(id) > 30 {
			id = "..." + id[len(id)-27:]
		}

		// Build simple info string (details in detail view)
		var infoParts []string
		if r.Scan.TotalHits() > 0 {
			infoParts = append(infoParts, fmt.Sprintf("Banned:%d", r.Scan.TotalHits()))
		}
		if r.Submission.HasMissingFiles() {
			infoParts = append(infoParts, fmt.Sprintf("Missing:%d", len(r.Submission.MissingFiles)))
		}
		infoStr := strings.Join(infoParts, " ")
		if infoStr == "" {
			infoStr = "-"
		}

		paddedStatus := fmt.Sprintf("%-4s", statusIcon)
		paddedId := fmt.Sprintf("%-30s", id)
		paddedCompile := fmt.Sprintf("%-7s", compileStr)

		var line string
		if i == m.cursor {
			plainLine := fmt.Sprintf("  %s  %s  %s  %s", paddedStatus, paddedId, paddedCompile, infoStr)
			line = styles.SelectedItem.Render(plainLine)
		} else {
			// Color the padded strings
			coloredStatus := paddedStatus
			switch statusStyle {
			case "success":
				coloredStatus = styles.SuccessStyle.Render(paddedStatus)
			case "warning":
				coloredStatus = styles.WarningStyle.Render(paddedStatus)
			case "error":
				coloredStatus = styles.ErrorStyle.Render(paddedStatus)
			}
			coloredCompile := paddedCompile
			switch compileStyle {
			case "success":
				coloredCompile = styles.SuccessStyle.Render(paddedCompile)
			case "warning":
				coloredCompile = styles.WarningStyle.Render(paddedCompile)
			case "error":
				coloredCompile = styles.ErrorStyle.Render(paddedCompile)
			}
			coloredInfo := infoStr
			if r.Scan.TotalHits() > 0 || r.Submission.HasMissingFiles() {
				coloredInfo = styles.WarningStyle.Render(infoStr)
			}
			line = fmt.Sprintf("  %s  %s  %s  %s", coloredStatus, paddedId, coloredCompile, coloredInfo)
		}
		table.WriteString(line)
		table.WriteString("\n")
	}

	// Scroll indicator
	if len(filtered) > m.visibleRows {
		table.WriteString(styles.Subtle.Render(fmt.Sprintf("\n  Showing %d-%d of %d",
			m.scrollOffset+1, endIdx, len(filtered))))
	}

	b.WriteString(tableBox.Render(table.String()))

	b.WriteString("\n\n")
	b.WriteString(m.renderHelpBar([]helpItem{
		{"↑/↓", "navigate"},
		{"enter", "details"},
		{"f", "filter"},
		{"r", "re-run"},
		{"e", "export"},
		{"esc", "back"},
	}))

	return b.String()
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

	// Content box
	contentBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(1, 2).
		Width(m.width - 4)

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
	b.WriteString(m.renderHelpBar([]helpItem{
		{"tab", "switch tab"},
		{"↑/↓", "scroll"},
		{"esc", "back"},
	}))

	return b.String()
}

func (m Model) renderCompileTab(r domain.SubmissionResult) string {
	var b strings.Builder

	if r.Compile.OK {
		b.WriteString(styles.SuccessStyle.Render("Compilation successful"))
	} else if r.Compile.TimedOut {
		b.WriteString(styles.ErrorStyle.Render("Compilation timed out (5s limit)"))
	} else {
		b.WriteString(styles.ErrorStyle.Render(fmt.Sprintf("Compilation failed (exit %d)", r.Compile.ExitCode)))
	}
	b.WriteString("\n\n")

	b.WriteString(styles.Subtle.Render("Command:"))
	b.WriteString("\n")
	cmd := strings.Join(r.Compile.Command, " ")
	if len(cmd) > 70 {
		cmd = cmd[:67] + "..."
	}
	b.WriteString(fmt.Sprintf("%s\n", cmd))

	if r.Compile.Stderr != "" {
		b.WriteString("\n")
		b.WriteString(styles.Subtle.Render("Output:"))
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
	}

	return b.String()
}

func (m Model) renderBannedTab(r domain.SubmissionResult) string {
	var b strings.Builder

	if r.Scan.TotalHits() == 0 {
		b.WriteString(styles.SuccessStyle.Render("No banned function calls detected"))
		return b.String()
	}

	b.WriteString(styles.WarningStyle.Render(fmt.Sprintf("%d banned call(s) found", r.Scan.TotalHits())))
	b.WriteString("\n\n")

	// Sort function names for stable display
	var funcNames []string
	for fn := range r.Scan.HitsByFunction {
		funcNames = append(funcNames, fn)
	}
	sort.Strings(funcNames)

	for _, fn := range funcNames {
		hits := r.Scan.HitsByFunction[fn]
		b.WriteString(styles.Highlight.Render(fmt.Sprintf("%s (%d)", fn, len(hits))))
		b.WriteString("\n")

		for _, hit := range hits {
			b.WriteString(styles.Subtle.Render(fmt.Sprintf("  %s:%d ", hit.File, hit.Line)))
			snippet := hit.Snippet
			if len(snippet) > 45 {
				snippet = snippet[:42] + "..."
			}
			b.WriteString(snippet)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderFilesTab(r domain.SubmissionResult) string {
	var b strings.Builder

	b.WriteString(styles.Subtle.Render(fmt.Sprintf("%d source file(s)", len(r.Submission.CFiles))))
	b.WriteString("\n\n")

	for _, f := range r.Submission.CFiles {
		b.WriteString(fmt.Sprintf("  %s\n", f))
	}

	if len(r.Scan.ParseErrors) > 0 {
		b.WriteString("\n")
		b.WriteString(styles.WarningStyle.Render("Parse errors:"))
		b.WriteString("\n")
		for _, e := range r.Scan.ParseErrors {
			b.WriteString(fmt.Sprintf("  - %s\n", e))
		}
	}

	return b.String()
}

func (m Model) renderHelp() string {
	var b strings.Builder

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		Padding(1, 2)

	b.WriteString(header.Render("Help - autOScan"))
	b.WriteString("\n\n")

	helpBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(1, 2).
		Width(70)

	help := `
WHAT IS AUTOSCAN?

autOScan is an automated C lab submission grader. It discovers student 
submissions, compiles them with gcc, and scans for banned function calls.

HOW TO USE:

1. Create a policy via Manage Policies (or edit ~/.config/autoscan/policies/)
2. Run autOScan and select your policy
3. Enter the path to your submissions folder
4. View results, filter, and export

CONFIG LOCATION:

  ~/.config/autoscan/
    policies/     Policy YAML files
    banned.txt    Global banned functions list

KEYBOARD SHORTCUTS:

  ↑/↓ or j/k   Navigate lists
  Enter        Select / View details
  Tab          Switch tabs in details view
  f            Cycle through filters
  r            Re-run the grader
  e            Export results
  Esc/q        Go back / Quit

EXPORT FORMATS:

  • Markdown - Human-readable grading notes
  • JSON     - Full structured data
  • CSV      - Spreadsheet-compatible
`

	b.WriteString(helpBox.Render(help))

	b.WriteString("\n\n")
	b.WriteString(m.renderHelpBar([]helpItem{
		{"esc", "back"},
	}))

	return b.String()
}

func (m Model) renderExport() string {
	var b strings.Builder

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		Padding(1, 2)

	b.WriteString(header.Render("Export Results"))
	b.WriteString("\n\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(1, 2).
		Width(50)

	var content strings.Builder
	content.WriteString("Select export format:\n\n")

	formats := []struct {
		name string
		desc string
	}{
		{"Markdown", "Human-readable report"},
		{"JSON", "Structured data"},
		{"CSV", "Spreadsheet format"},
	}

	for i, f := range formats {
		cursor := "  "
		style := styles.NormalItem
		if i == m.exportCursor {
			cursor = "▸ "
			style = styles.SelectedItem
		}
		content.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(f.name)))
		if i == m.exportCursor {
			content.WriteString(styles.Subtle.Render(fmt.Sprintf("    %s\n", f.desc)))
		}
	}

	b.WriteString(box.Render(content.String()))

	b.WriteString("\n\n")
	b.WriteString(styles.Subtle.Render("  Files will be exported to current directory"))
	b.WriteString("\n\n")
	b.WriteString(m.renderHelpBar([]helpItem{
		{"↑/↓", "navigate"},
		{"enter", "export"},
		{"esc", "back"},
	}))

	return b.String()
}

type helpItem struct {
	key  string
	desc string
}

func (m Model) renderHelpBar(items []helpItem) string {
	var parts []string
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s %s",
			styles.HelpKey.Render(item.key),
			styles.HelpDesc.Render(item.desc)))
	}
	return "  " + strings.Join(parts, "  •  ")
}

// Start initializes and runs the TUI
func Start(cfg Config) error {
	p := tea.NewProgram(New(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
