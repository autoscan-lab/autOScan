// Package tui provides the terminal user interface for autOScan.
package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/felipetrejos/autoscan/internal/config"
	"github.com/felipetrejos/autoscan/internal/domain"
	"github.com/felipetrejos/autoscan/internal/engine"
	"github.com/felipetrejos/autoscan/internal/policy"
	"github.com/felipetrejos/autoscan/internal/tui/components"
	"github.com/felipetrejos/autoscan/internal/tui/styles"
)

// ─────────────────────────────────────────────────────────────────────────────
// View Types
// ─────────────────────────────────────────────────────────────────────────────

// View represents different screens in the app
type View int

const (
	ViewHome View = iota
	ViewPolicySelect
	ViewPolicyManage
	ViewPolicyEditor
	ViewBannedEditor
	ViewSettings
	ViewDirectoryInput
	ViewSubmissions
	ViewDetails
	ViewExport
)

// ─────────────────────────────────────────────────────────────────────────────
// Constants
// ─────────────────────────────────────────────────────────────────────────────

const (
	// Minimum terminal dimensions
	minWidth  = 60
	minHeight = 20
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

// MenuItem for home screen
type MenuItem int

const (
	MenuRunGrader MenuItem = iota
	MenuManagePolicies
	MenuSettings
	MenuUninstall
	MenuQuit
)

// ─────────────────────────────────────────────────────────────────────────────
// Messages
// ─────────────────────────────────────────────────────────────────────────────

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
	uninstallDoneMsg    struct{}
	bannedListLoadedMsg []string
	bannedListSavedMsg  struct{}

	// Execution messages
	executeResultMsg struct {
		result domain.ExecuteResult
	}
	executeTestResultsMsg struct {
		results []domain.ExecuteResult
	}
	multiProcessResultMsg struct {
		result *domain.MultiProcessResult
	}
)

// ─────────────────────────────────────────────────────────────────────────────
// Model
// ─────────────────────────────────────────────────────────────────────────────

// Model is the main Bubble Tea model
type Model struct {
	// View state
	currentView View
	width       int
	height      int

	// Settings
	settings       config.Settings
	settingsCursor int

	// Home menu
	menuItem MenuItem

	// Animation
	eyeAnimation components.EyeAnimation
	helpPanel    components.HelpPanel

	// Policy selection
	policies       []*policy.Policy
	selectedPolicy int

	// Policy management
	policyManageCursor int
	policyEditor       PolicyEditor
	confirmDelete      bool

	// Banned list editor
	bannedList       []string
	bannedCursorEdit int
	bannedInput      textinput.Model
	bannedEditing    bool

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
	detailsTab    int
	detailScroll  int
	bannedCursor  int
	expandedFuncs map[string]bool

	// Run tab state
	runArgsInput       textinput.Model
	runStdinInput      textinput.Model
	runInputFocused    int // 0 = args, 1 = stdin, 2 = run button, 3 = test cases, 4+ = multi-process
	runResult          *domain.ExecuteResult
	runTestResults     []domain.ExecuteResult
	runTestCursor      int
	isExecuting        bool
	executor           *engine.Executor
	multiProcessResult *domain.MultiProcessResult
	showMultiProcess   bool
	runCancelFunc      context.CancelFunc // To cancel/kill running processes

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

	// Initialize text input for banned editor
	bannedInput := textinput.New()
	bannedInput.Placeholder = "function_name"
	bannedInput.CharLimit = 64

	// Initialize text inputs for run tab
	runArgsInput := textinput.New()
	runArgsInput.Placeholder = "arg1 arg2 arg3..."
	runArgsInput.CharLimit = 256
	runArgsInput.Width = 40

	runStdinInput := textinput.New()
	runStdinInput.Placeholder = "stdin input (use \\n for newlines)"
	runStdinInput.CharLimit = 1024
	runStdinInput.Width = 40

	// Load settings
	settings, _ := config.LoadSettings()

	// Initialize help panel
	helpPanel := components.NewHelpPanel(28, styles.Version)

	return Model{
		currentView:   ViewHome,
		width:         minWidth,
		height:        minHeight,
		settings:      settings,
		root:          root,
		spinner:       s,
		folderBrowser: NewFolderBrowser(root),
		visibleRows:   20,
		filter:        FilterAll,
		menuItem:      MenuRunGrader,
		policyEditor:  NewPolicyEditor(80, 40),
		bannedInput:   bannedInput,
		runArgsInput:  runArgsInput,
		runStdinInput: runStdinInput,
		eyeAnimation:  components.NewEyeAnimation(),
		helpPanel:     helpPanel,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadPolicies(),
		m.spinner.Tick,
		m.eyeAnimation.Init(),
	)
}

// Start initializes and runs the TUI
func Start(cfg Config) error {
	p := tea.NewProgram(New(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
