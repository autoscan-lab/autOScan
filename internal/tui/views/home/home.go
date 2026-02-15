package home

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/felitrejos/autoscan/internal/tui/components"
)

const (
	MenuRunGrader = iota
	MenuManagePolicies
	MenuSettings
	MenuUninstall
	MenuQuit
)

const logo = `
 █████╗ ██╗   ██╗████████╗ ██████╗ ███████╗ ██████╗ █████╗ ███╗   ██╗
██╔══██╗██║   ██║╚══██╔══╝██╔═══██╗██╔════╝██╔════╝██╔══██╗████╗  ██║
███████║██║   ██║   ██║   ██║   ██║███████╗██║     ███████║██╔██╗ ██║
██╔══██║██║   ██║   ██║   ██║   ██║╚════██║██║     ██╔══██║██║╚██╗██║
██║  ██║╚██████╔╝   ██║   ╚██████╔╝███████║╚██████╗██║  ██║██║ ╚████║
╚═╝  ╚═╝ ╚═════╝    ╚═╝    ╚═════╝ ╚══════╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═══╝`

const tagline = "OS Lab Submission Grader"

type State struct {
	Width         int
	MenuWidth     int
	MenuItem      int
	ConfirmDelete bool
	PolicyCount   int
	AnimationView string
	HelpPanelView string
}

type Navigation int

const (
	NavNone Navigation = iota
	NavPolicySelect
	NavPolicyManage
	NavSettings
	NavQuit
	NavUninstall
)

type UpdateResult struct {
	MenuItem                int
	ConfirmDelete           bool
	Navigation              Navigation
	ResetPolicyManageCursor bool
	ResetSettingsCursor     bool
}

func View(s State) string {
	var b strings.Builder

	menuWidth := s.MenuWidth
	if menuWidth < 45 {
		menuWidth = 45
	}

	logoStyled := renderGradientLogo(logo)
	taglineStyled := components.SubtleText.Render("     " + tagline)
	animationBox := lipgloss.NewStyle().
		Width(20).
		Align(lipgloss.Center).
		Render(s.AnimationView)
	logoWithTagline := logoStyled + "\n" + taglineStyled

	topSection := lipgloss.JoinHorizontal(
		lipgloss.Top,
		logoWithTagline,
		lipgloss.NewStyle().PaddingLeft(4).Render(animationBox),
	)
	topWidth := lipgloss.Width(topSection)
	maxMenuWidthForHeader := topWidth - 2
	if maxMenuWidthForHeader >= 45 && menuWidth > maxMenuWidthForHeader {
		menuWidth = maxMenuWidthForHeader
	}

	b.WriteString(topSection)
	b.WriteString("\n\n")

	menuBox := components.PrimaryBoxStyle().
		Padding(1, 3).
		Width(menuWidth)

	var menu strings.Builder
	menuItems := []struct {
		key  string
		desc string
		item int
	}{
		{"1", "Run Grader", MenuRunGrader},
		{"2", "Manage Policies", MenuManagePolicies},
		{"3", "Settings", MenuSettings},
		{"4", "Uninstall", MenuUninstall},
		{"q", "Quit", MenuQuit},
	}

	for _, mi := range menuItems {
		cursor := "  "
		style := components.NormalItem
		if mi.item == s.MenuItem {
			cursor = "▸ "
			style = components.SelectedItem
		}
		keyStyle := components.HelpKey.Render(fmt.Sprintf("[%s]", mi.key))
		menu.WriteString(fmt.Sprintf("%s%s %s\n", cursor, keyStyle, style.Render(mi.desc)))
	}

	if s.ConfirmDelete && s.MenuItem == MenuUninstall {
		menu.WriteString("\n")
		menu.WriteString(components.ConfirmDialog("Remove autoscan and all configs?"))
	}

	b.WriteString(s.HelpPanelView)
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().MarginTop(1).Render(menuBox.Render(menu.String())))
	b.WriteString("\n\n")
	b.WriteString(components.SubtleText.Render("  Use ↑/↓ to navigate, Enter to select"))

	return b.String()
}

func Update(s State, msg tea.KeyMsg) UpdateResult {
	result := UpdateResult{
		MenuItem:      s.MenuItem,
		ConfirmDelete: s.ConfirmDelete,
		Navigation:    NavNone,
	}

	switch msg.String() {
	case "j", "down":
		if result.MenuItem < MenuQuit {
			result.MenuItem++
		}
	case "k", "up":
		if result.MenuItem > MenuRunGrader {
			result.MenuItem--
		}
	case "enter":
		switch result.MenuItem {
		case MenuRunGrader:
			result.Navigation = NavPolicySelect
		case MenuManagePolicies:
			result.Navigation = NavPolicyManage
			result.ResetPolicyManageCursor = true
		case MenuSettings:
			result.Navigation = NavSettings
			result.ResetSettingsCursor = true
		case MenuUninstall:
			result.ConfirmDelete = true
		case MenuQuit:
			result.Navigation = NavQuit
		}
	case "y":
		if result.ConfirmDelete && result.MenuItem == MenuUninstall {
			result.Navigation = NavUninstall
		}
	case "n", "esc":
		result.ConfirmDelete = false
	case "q":
		if !result.ConfirmDelete {
			result.Navigation = NavQuit
		} else {
			result.ConfirmDelete = false
		}
	case "1":
		result.Navigation = NavPolicySelect
	case "2":
		result.Navigation = NavPolicyManage
		result.ResetPolicyManageCursor = true
	case "3":
		result.Navigation = NavSettings
		result.ResetSettingsCursor = true
	case "4":
		result.ConfirmDelete = true
		result.MenuItem = MenuUninstall
	}

	return result
}

type rgb struct {
	r int
	g int
	b int
}

func renderGradientLogo(logoText string) string {
	lines := strings.Split(strings.Trim(logoText, "\n"), "\n")
	if len(lines) == 0 {
		return ""
	}

	top := rgb{r: 147, g: 197, b: 253}
	mid := rgb{r: 96, g: 165, b: 250}
	bottom := rgb{r: 30, g: 64, b: 175}

	var b strings.Builder
	last := len(lines) - 1
	for i, line := range lines {
		t := 0.0
		if last > 0 {
			t = float64(i) / float64(last)
		}

		var c rgb
		if t <= 0.5 {
			c = mix(top, mid, t*2)
		} else {
			c = mix(mid, bottom, (t-0.5)*2)
		}

		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", c.r, c.g, c.b)))
		b.WriteString(style.Render(line))
		if i < last {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

func mix(a, b rgb, t float64) rgb {
	return rgb{
		r: int(float64(a.r) + (float64(b.r)-float64(a.r))*t),
		g: int(float64(a.g) + (float64(b.g)-float64(a.g))*t),
		b: int(float64(a.b) + (float64(b.b)-float64(a.b))*t),
	}
}
