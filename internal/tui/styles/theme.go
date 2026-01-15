package styles

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	Primary   = lipgloss.Color("#7C3AED") // Purple
	Secondary = lipgloss.Color("#10B981") // Green
	Muted     = lipgloss.Color("#6B7280") // Gray
	Error     = lipgloss.Color("#EF4444") // Red
)

// Base styles
var (
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary)

	Subtle = lipgloss.NewStyle().
		Foreground(Muted)

	Highlight = lipgloss.NewStyle().
			Bold(true).
			Foreground(Secondary)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Error)

	Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Muted).
		Padding(1, 2)
)
