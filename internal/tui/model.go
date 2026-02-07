package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/felitrejos/autoscan/internal/config"
	"github.com/felitrejos/autoscan/internal/domain"
	"github.com/felitrejos/autoscan/internal/engine"
	"github.com/felitrejos/autoscan/internal/policy"
	"github.com/felitrejos/autoscan/internal/tui/components"
	policyview "github.com/felitrejos/autoscan/internal/tui/views/policy"
)

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

const (
	minWidth  = 60
	minHeight = 20
)

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

type MenuItem int

const (
	MenuRunGrader MenuItem = iota
	MenuManagePolicies
	MenuSettings
	MenuUninstall
	MenuQuit
)

type (
	policiesLoadedMsg    []*policy.Policy
	discoveryCompleteMsg []domain.Submission
	compileCompleteMsg   struct {
		sub    domain.Submission
		result domain.CompileResult
	}
	runCompleteMsg        domain.RunReport
	errorMsg              error
	exportDoneMsg         struct{ format, path string }
	uninstallDoneMsg      struct{}
	bannedListLoadedMsg   []string
	bannedListSavedMsg    struct{}
	executeResultMsg      struct{ result domain.ExecuteResult }
	executeTestResultsMsg struct{ results []domain.ExecuteResult }
	multiProcessResultMsg struct{ result *domain.MultiProcessResult }
	multiProcessUpdateMsg struct{ result *domain.MultiProcessResult }
	multiProcessTickMsg   struct{}
	similarityStartedMsg  struct {
		process string
		runID   int64
	}
	similarityComputedMsg struct {
		process string
		pairs   []SimilarityPair
		runID   int64
	}
	similarityErrorMsg struct {
		process string
		err     error
		runID   int64
	}
	pairDetailLoadedMsg struct {
		process   string
		pairIndex int
		contentA  []byte
		contentB  []byte
		err       error
		runID     int64
	}
)

type SimilarityComputeState int

const (
	SimilarityNotStarted SimilarityComputeState = iota
	SimilarityComputing
	SimilarityDone
	SimilarityError
)

type SimilarityPair = domain.SimilarityPairResult

type Model struct {
	currentView View
	width       int
	height      int

	settings       config.Settings
	settingsCursor int
	menuItem       MenuItem

	eyeAnimation components.EyeAnimation
	helpPanel    components.HelpPanel

	policies           []*policy.Policy
	selectedPolicy     int
	policyManageCursor int
	policyEditor       policyview.Editor
	confirmDelete      bool

	bannedList       []string
	bannedCursorEdit int
	bannedInput      textinput.Model
	bannedEditing    bool

	folderBrowser components.FolderBrowser
	root          string
	inputError    string

	runner      *engine.Runner
	submissions []domain.Submission
	results     []domain.SubmissionResult
	report      *domain.RunReport
	isRunning   bool
	completed   int
	spinner     spinner.Model
	runError    string
	runID       int64

	cursor       int
	scrollOffset int
	visibleRows  int
	filter       Filter
	searchInput  textinput.Model
	searchActive bool
	searchQuery  string

	detailsTab    int
	detailScroll  int
	bannedCursor  int
	expandedFuncs map[string]bool

	runArgsInput           textinput.Model
	runStdinInput          textinput.Model
	runInputFocused        int
	runResult              *domain.ExecuteResult
	runTestResults         []domain.ExecuteResult
	isExecuting            bool
	executor               *engine.Executor
	multiProcessResult     *domain.MultiProcessResult
	showMultiProcess       bool
	runCancelFunc          context.CancelFunc
	multiProcessUpdateChan <-chan *domain.MultiProcessResult
	outputScroll           int
	selectedProcessIdx     int

	submissionsTab           int
	similarityProcessNames   []string
	similaritySelectedProc   int
	similarityPairsByProcess map[string][]SimilarityPair
	similarityStateByProcess map[string]SimilarityComputeState
	similarityErrorByProcess map[string]string
	similarityInFlight       map[string]bool
	similarityCursor         int
	similarityScroll         int

	pairDetailOpen        bool
	pairDetailProcess     string
	pairDetailPairIndex   int
	pairDetailContentA    []byte
	pairDetailContentB    []byte
	pairDetailLoadErr     string
	pairDetailFocusedPane int // 0 = left (A), 1 = right (B)
	pairDetailScrollA     int
	pairDetailScrollB     int
	pairDetailHScrollA    int
	pairDetailHScrollB    int

	exportCursor int
	statusMsg    string
}

func (m *Model) resetPairDetailViewState() {
	m.pairDetailContentA = nil
	m.pairDetailContentB = nil
	m.pairDetailLoadErr = ""
	m.pairDetailFocusedPane = 0
	m.pairDetailScrollA = 0
	m.pairDetailScrollB = 0
	m.pairDetailHScrollA = 0
	m.pairDetailHScrollB = 0
}

func (m *Model) resetPairDetailState() {
	m.pairDetailOpen = false
	m.pairDetailProcess = ""
	m.pairDetailPairIndex = 0
	m.resetPairDetailViewState()
}

type Config struct {
	Root string
}

func New(cfg Config) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(components.Primary)

	root := cfg.Root
	if root == "" {
		root = "."
	}

	bannedInput := textinput.New()
	bannedInput.Placeholder = "function_name"
	bannedInput.CharLimit = 64

	runArgsInput := textinput.New()
	runArgsInput.Placeholder = "arg1 arg2 arg3..."
	runArgsInput.CharLimit = 256
	runArgsInput.Width = 40

	runStdinInput := textinput.New()
	runStdinInput.Placeholder = "stdin input (use \\n for newlines)"
	runStdinInput.CharLimit = 1024
	runStdinInput.Width = 40

	searchInput := textinput.New()
	searchInput.Placeholder = "search student by name..."
	searchInput.CharLimit = 64
	searchInput.Prompt = ""
	searchInput.Width = 30

	settings, _ := config.LoadSettings()
	helpPanel := components.NewHelpPanel(28, components.Version)

	return Model{
		currentView:              ViewHome,
		width:                    minWidth,
		height:                   minHeight,
		settings:                 settings,
		root:                     root,
		spinner:                  s,
		folderBrowser:            components.NewFolderBrowser(root),
		visibleRows:              20,
		filter:                   FilterAll,
		searchInput:              searchInput,
		menuItem:                 MenuRunGrader,
		policyEditor:             policyview.NewEditor(80, 40),
		bannedInput:              bannedInput,
		runArgsInput:             runArgsInput,
		runStdinInput:            runStdinInput,
		eyeAnimation:             components.NewEyeAnimation(),
		helpPanel:                helpPanel,
		submissionsTab:           0,
		similaritySelectedProc:   0,
		similarityPairsByProcess: make(map[string][]SimilarityPair),
		similarityStateByProcess: make(map[string]SimilarityComputeState),
		similarityErrorByProcess: make(map[string]string),
		similarityInFlight:       make(map[string]bool),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadPolicies(), m.spinner.Tick, m.eyeAnimation.Init())
}

func (m *Model) clearRunResults() {
	m.runResult = nil
	m.runTestResults = nil
	m.multiProcessResult = nil
	m.showMultiProcess = false
}

func (m *Model) openPolicyEditor(p *policy.Policy) tea.Cmd {
	m.policyEditor.Reset()
	if p != nil {
		m.policyEditor.LoadPolicy(p)
	}
	m.policyEditor.SetWidth(m.width)
	m.currentView = ViewPolicyEditor
	return textinput.Blink
}

func Start(cfg Config) error {
	p := tea.NewProgram(New(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
