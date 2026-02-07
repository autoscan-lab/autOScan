package directory

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felitrejos/autoscan/internal/tui/components"
)

type State struct {
	Width         int
	InputError    string
	FolderBrowser components.FolderBrowser
}

type UpdateResult struct {
	GoBack        bool
	Selected      bool
	SelectedPath  string
	FolderBrowser components.FolderBrowser
	Cmd           tea.Cmd
}

func View(s State) string {
	var b strings.Builder

	b.WriteString(components.RenderHeader("Select Directory"))

	boxWidth := components.BoxWidth(s.Width, 8, 60)
	box := components.BoxStyle(boxWidth)

	var content strings.Builder
	content.WriteString(components.SubtleText.Render("Navigate to submissions folder"))
	content.WriteString("\n\n")
	content.WriteString(s.FolderBrowser.View())

	b.WriteString(box.Render(content.String()))

	if s.InputError != "" {
		b.WriteString("\n")
		b.WriteString(components.ErrorText.Render("  " + s.InputError))
	}

	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "select/open"},
		{Key: "←/backspace", Desc: "parent dir"},
		{Key: "esc", Desc: "cancel"},
	}))

	return b.String()
}

func Update(s State, msg tea.KeyMsg) UpdateResult {
	result := UpdateResult{
		FolderBrowser: s.FolderBrowser,
	}

	if msg.String() == "esc" {
		result.GoBack = true
		return result
	}

	selected, cmd := result.FolderBrowser.Update(msg)
	result.Cmd = cmd
	if selected {
		result.Selected = true
		result.SelectedPath = result.FolderBrowser.Selected()
	}

	return result
}
