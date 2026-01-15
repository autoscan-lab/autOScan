package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// View represents different screens in the app
type View int

const (
	ViewHome View = iota
	ViewIndex
	ViewSearch
	ViewChat
)

// Model is the main Bubble Tea model
type Model struct {
	currentView View
	width       int
	height      int

	// Sub-models for each view
	// homeModel   homeModel
	// indexModel  indexModel
	// searchModel searchModel
	// chatModel   chatModel
}

// New creates a new TUI model
func New() Model {
	return Model{
		currentView: ViewHome,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.currentView = ViewIndex
		case "2":
			m.currentView = ViewSearch
		case "3":
			m.currentView = ViewChat
		case "esc":
			m.currentView = ViewHome
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m Model) View() string {
	switch m.currentView {
	case ViewIndex:
		return m.renderIndex()
	case ViewSearch:
		return m.renderSearch()
	case ViewChat:
		return m.renderChat()
	default:
		return m.renderHome()
	}
}

func (m Model) renderHome() string {
	return fmt.Sprintf(`
  ╭─────────────────────────────────────╮
  │           f e l i t u i v e         │
  │   Local-first RAG in your terminal  │
  ╰─────────────────────────────────────╯

  [1] Index      Index a folder
  [2] Search     Semantic search
  [3] Chat       Chat with your files

  [q] Quit

`)
}

func (m Model) renderIndex() string {
	return `
  Index View (TODO)

  Press [esc] to go back
`
}

func (m Model) renderSearch() string {
	return `
  Search View (TODO)

  Press [esc] to go back
`
}

func (m Model) renderChat() string {
	return `
  Chat View (TODO)

  Press [esc] to go back
`
}

// Start initializes and runs the TUI
func Start() error {
	p := tea.NewProgram(New(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
