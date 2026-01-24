package styles

import "github.com/charmbracelet/lipgloss"

// Application version
const Version = "2.0.0"

// ─────────────────────────────────────────────────────────────────────────────
// Color Palette
// ─────────────────────────────────────────────────────────────────────────────

var (
	// Primary colors
	Primary     = lipgloss.Color("#3B82F6") // Blue - main accent
	PrimaryDim  = lipgloss.Color("#1D4ED8") // Darker blue for backgrounds
	PrimaryGlow = lipgloss.Color("#60A5FA") // Lighter blue for highlights

	// Secondary/Accent colors
	Accent    = lipgloss.Color("#06B6D4") // Cyan - secondary accent
	AccentDim = lipgloss.Color("#0891B2") // Darker cyan

	// Semantic colors
	Success = lipgloss.Color("#22C55E") // Green
	Warning = lipgloss.Color("#F59E0B") // Amber
	Error   = lipgloss.Color("#EF4444") // Red
	Danger  = lipgloss.Color("#EF4444") // Red (alias)

	// Neutral colors
	Muted       = lipgloss.Color("#6B7280") // Gray for subtle text
	SubtleColor = lipgloss.Color("#9CA3AF") // Lighter gray
	Text        = lipgloss.Color("#E5E7EB") // Main text
	TextBright  = lipgloss.Color("#F9FAFB") // Bright text

	// Background colors
	BgDark     = lipgloss.Color("#0F172A") // Darkest background
	BgPanel    = lipgloss.Color("#1E293B") // Panel background
	BgHover    = lipgloss.Color("#334155") // Hover/selected background
	BgHighlight = lipgloss.Color("#1E3A5F") // Highlighted items
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
var (
	ErrorStyle   = ErrorText
	WarningStyle = WarningText
	SuccessStyle = SuccessText
)

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

	// HoveredItem for items under cursor but not selected
	HoveredItem = lipgloss.NewStyle().
			Background(BgHover).
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

// AccentBoxStyle creates a box with accent-colored border
func AccentBoxStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Primary).
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

var (
	// HeaderStyle for page/section headers
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			Padding(1, 2)

	// TitleStyle for major titles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryGlow)
)

// ─────────────────────────────────────────────────────────────────────────────
// Status Indicator Styles
// ─────────────────────────────────────────────────────────────────────────────

var (
	// StatusClean for clean/passing items
	StatusClean = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	// StatusWarning for items with warnings
	StatusWarning = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)

	// StatusError for failed/error items
	StatusError = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	// StatusMuted for pending/inactive items
	StatusMuted = lipgloss.NewStyle().
			Foreground(Muted)
)

// ─────────────────────────────────────────────────────────────────────────────
// Logo Style
// ─────────────────────────────────────────────────────────────────────────────

var (
	// LogoStyle for the main application logo
	LogoStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	// LogoAccentStyle for accent parts of logo
	LogoAccentStyle = lipgloss.NewStyle().
			Foreground(Accent).
			Bold(true)
)

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
