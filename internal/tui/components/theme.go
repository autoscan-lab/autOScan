package components

import "github.com/charmbracelet/lipgloss"

const Version = "3.0.0"

var (
	Primary     = lipgloss.Color("#3B82F6")
	PrimaryGlow = lipgloss.Color("#60A5FA")
	Accent      = lipgloss.Color("#06B6D4")
	Success     = lipgloss.Color("#22C55E")
	Warning     = lipgloss.Color("#F59E0B")
	Error       = lipgloss.Color("#EF4444")
	Muted       = lipgloss.Color("#6B7280")
	Text        = lipgloss.Color("#E5E7EB")
	BgPanel     = lipgloss.Color("#1E293B")
)

var (
	SubtleText  = lipgloss.NewStyle().Foreground(Muted)
	Highlight   = lipgloss.NewStyle().Bold(true).Foreground(Accent)
	ErrorText   = lipgloss.NewStyle().Foreground(Error)
	WarningText = lipgloss.NewStyle().Foreground(Warning)
	SuccessText = lipgloss.NewStyle().Foreground(Success)
	PrimaryText = lipgloss.NewStyle().Foreground(Primary)
)

var ErrorStyle = ErrorText

var (
	SelectedItem = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	NormalItem   = lipgloss.NewStyle().Foreground(Text)
)

var (
	TabActive   = lipgloss.NewStyle().Bold(true).Foreground(Primary).Background(BgPanel).Padding(0, 2)
	TabInactive = lipgloss.NewStyle().Foreground(Muted).Padding(0, 2)
)

var (
	HelpKey  = lipgloss.NewStyle().Foreground(PrimaryGlow).Bold(true)
	HelpDesc = lipgloss.NewStyle().Foreground(Muted)
)

func RoundedBox() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Muted).
		Padding(1, 2)
}

func BoxStyle(width int) lipgloss.Style {
	return RoundedBox().Width(width)
}

func WarningBoxStyle(width int) lipgloss.Style {
	return RoundedBox().BorderForeground(Warning).Width(width)
}

func ErrorBoxStyle() lipgloss.Style {
	return RoundedBox().BorderForeground(Error)
}

func PrimaryBoxStyle() lipgloss.Style {
	return RoundedBox().BorderForeground(Primary)
}

func TableBoxStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Muted).
		Padding(0, 1)
}

func CompactBoxStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Muted).
		Padding(0, 2)
}

func FormBoxStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Muted).
		Padding(0, 1)
}

func FixedWidthStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().Width(width).MaxWidth(width)
}

var HeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(Primary).Padding(1, 2)

var LogoStyle = lipgloss.NewStyle().Foreground(Primary).Bold(true)

var (
	EyeColor      = lipgloss.NewStyle().Foreground(PrimaryGlow)
	EyePupilColor = lipgloss.NewStyle().Foreground(Accent)
)

var Subtle = SubtleText
