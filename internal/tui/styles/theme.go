package styles

import "github.com/charmbracelet/lipgloss"

// Application version
const Version = "2.1.0"

// ─────────────────────────────────────────────────────────────────────────────
// Color Palette
// ─────────────────────────────────────────────────────────────────────────────

var (
	// Primary colors
	Primary     = lipgloss.Color("#3B82F6") // Blue - main accent
	PrimaryGlow = lipgloss.Color("#60A5FA") // Lighter blue for highlights

	// Secondary/Accent colors
	Accent = lipgloss.Color("#06B6D4") // Cyan - secondary accent

	// Semantic colors
	Success = lipgloss.Color("#22C55E") // Green
	Warning = lipgloss.Color("#F59E0B") // Amber
	Error   = lipgloss.Color("#EF4444") // Red

	// Neutral colors
	Muted = lipgloss.Color("#6B7280") // Gray for subtle text
	Text  = lipgloss.Color("#E5E7EB") // Main text

	// Background colors
	BgPanel = lipgloss.Color("#1E293B") // Panel background
)

// ─────────────────────────────────────────────────────────────────────────────
// Text Styles
// ─────────────────────────────────────────────────────────────────────────────

var (
	// SubtleText for less important information
	SubtleText = lipgloss.NewStyle().Foreground(Muted)

	// Highlight for emphasized text
	Highlight = lipgloss.NewStyle().
			Bold(true).
			Foreground(Accent)

	// ErrorText for error messages
	ErrorText = lipgloss.NewStyle().Foreground(Error)

	// WarningText for warning messages
	WarningText = lipgloss.NewStyle().Foreground(Warning)

	// SuccessText for success messages
	SuccessText = lipgloss.NewStyle().Foreground(Success)

	// PrimaryText for primary colored text
	PrimaryText = lipgloss.NewStyle().Foreground(Primary)
)

// Aliases for backward compatibility
var ErrorStyle = ErrorText

// ─────────────────────────────────────────────────────────────────────────────
// Interactive Item Styles
// ─────────────────────────────────────────────────────────────────────────────

var (
	// SelectedItem for currently selected list items
	SelectedItem = lipgloss.NewStyle().
			Foreground(Accent).
			Bold(true)

	// NormalItem for non-selected list items
	NormalItem = lipgloss.NewStyle().
			Foreground(Text)
)

// ─────────────────────────────────────────────────────────────────────────────
// Tab Styles
// ─────────────────────────────────────────────────────────────────────────────

var (
	// TabActive for the currently active tab
	TabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			Background(BgPanel).
			Padding(0, 2)

	// TabInactive for inactive tabs
	TabInactive = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(0, 2)
)

// ─────────────────────────────────────────────────────────────────────────────
// Help Bar Styles
// ─────────────────────────────────────────────────────────────────────────────

var (
	// HelpKey for keyboard shortcut keys
	HelpKey = lipgloss.NewStyle().
		Foreground(PrimaryGlow).
		Bold(true)

	// HelpDesc for shortcut descriptions
	HelpDesc = lipgloss.NewStyle().
		Foreground(Muted)
)

// ─────────────────────────────────────────────────────────────────────────────
// Box/Panel Styles
// ─────────────────────────────────────────────────────────────────────────────

// BoxStyle creates a standard bordered box
func BoxStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Muted).
		Padding(1, 2).
		Width(width)
}

// WarningBoxStyle creates a box with warning-colored border
func WarningBoxStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Warning).
		Padding(1, 2).
		Width(width)
}

// ─────────────────────────────────────────────────────────────────────────────
// Header Styles
// ─────────────────────────────────────────────────────────────────────────────

// HeaderStyle for page/section headers
var HeaderStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(Primary).
	Padding(1, 2)

// ─────────────────────────────────────────────────────────────────────────────
// Logo Style
// ─────────────────────────────────────────────────────────────────────────────

// LogoStyle for the main application logo
var LogoStyle = lipgloss.NewStyle().
	Foreground(Primary).
	Bold(true)

// ─────────────────────────────────────────────────────────────────────────────
// Animation Frame Colors (for eye animation)
// ─────────────────────────────────────────────────────────────────────────────

var (
	// EyeColor for the eye animation
	EyeColor = lipgloss.NewStyle().Foreground(PrimaryGlow)

	// EyePupilColor for the pupil
	EyePupilColor = lipgloss.NewStyle().Foreground(Accent)
)

// ─────────────────────────────────────────────────────────────────────────────
// Helper for backward compatibility
// ─────────────────────────────────────────────────────────────────────────────

// Subtle is an alias for SubtleText (backward compatibility)
var Subtle = SubtleText
