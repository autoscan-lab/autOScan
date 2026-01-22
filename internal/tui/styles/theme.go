package styles

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	Primary = lipgloss.Color("#7C3AED") // Purple
	Muted   = lipgloss.Color("#6B7280") // Gray
	Error   = lipgloss.Color("#EF4444") // Red
	Danger  = lipgloss.Color("#EF4444") // Red
	Warning = lipgloss.Color("#F59E0B") // Amber
	Success = lipgloss.Color("#22C55E") // Green
)

// Base styles
var (
	Subtle = lipgloss.NewStyle().Foreground(Muted)

	Highlight = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#10B981"))

	ErrorStyle   = lipgloss.NewStyle().Foreground(Error)
	WarningStyle = lipgloss.NewStyle().Foreground(Warning)
	SuccessStyle = lipgloss.NewStyle().Foreground(Success)

	SelectedItem = lipgloss.NewStyle().
			Background(lipgloss.Color("#374151")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	NormalItem = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB")).
			Padding(0, 1)

	TabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 2)

	TabInactive = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(0, 2)

	HelpKey  = lipgloss.NewStyle().Foreground(Primary).Bold(true)
	HelpDesc = lipgloss.NewStyle().Foreground(Muted)
)
