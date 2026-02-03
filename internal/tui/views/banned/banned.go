package banned

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/feli05/autoscan/internal/tui/components"
)

type State struct {
	Width            int
	BannedList       []string
	BannedCursorEdit int
	BannedEditing    bool
	BannedInput      textinput.Model
}

type UpdateResult struct {
	BannedList       []string
	BannedCursorEdit int
	BannedEditing    bool
	BannedInput      textinput.Model
	GoBack           bool
	Save             bool
	NeedsInputCmd    bool
}

func View(s State) string {
	var b strings.Builder

	b.WriteString(components.RenderHeader("Edit Banned Functions"))

	box := components.BoxStyle(min(50, s.Width-4))

	var content strings.Builder
	content.WriteString(components.SubtleText.Render("Global banned function list"))
	content.WriteString("\n\n")

	if len(s.BannedList) == 0 {
		content.WriteString(components.SubtleText.Render("  (no banned functions)"))
		content.WriteString("\n")
	} else {
		for i, fn := range s.BannedList {
			cursor := "  "
			style := components.NormalItem
			if i == s.BannedCursorEdit {
				cursor = "▸ "
				style = components.SelectedItem
			}

			if s.BannedEditing && i == s.BannedCursorEdit {
				content.WriteString(fmt.Sprintf("%s%s\n", cursor, s.BannedInput.View()))
			} else {
				content.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(fn)))
			}
		}
	}

	b.WriteString(box.Render(content.String()))

	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "a", Desc: "add"},
		{Key: "e/enter", Desc: "edit"},
		{Key: "d", Desc: "delete"},
		{Key: "esc", Desc: "save & exit"},
	}))

	return b.String()
}

func Update(s State, msg tea.KeyMsg) UpdateResult {
	result := UpdateResult{
		BannedList:       s.BannedList,
		BannedCursorEdit: s.BannedCursorEdit,
		BannedEditing:    s.BannedEditing,
		BannedInput:      s.BannedInput,
	}

	if s.BannedEditing {
		switch msg.String() {
		case "enter":
			newVal := strings.TrimSpace(s.BannedInput.Value())
			if newVal != "" && s.BannedCursorEdit < len(result.BannedList) {
				result.BannedList[s.BannedCursorEdit] = newVal
			}
			result.BannedEditing = false
			result.BannedInput.Blur()
		case "esc":
			result.BannedEditing = false
			result.BannedInput.Blur()
		default:
			var cmd tea.Cmd
			result.BannedInput, cmd = result.BannedInput.Update(msg)
			result.NeedsInputCmd = cmd != nil
		}
		return result
	}

	switch msg.String() {
	case "j", "down":
		if len(result.BannedList) > 0 && result.BannedCursorEdit < len(result.BannedList)-1 {
			result.BannedCursorEdit++
		}
	case "k", "up":
		if result.BannedCursorEdit > 0 {
			result.BannedCursorEdit--
		}
	case "enter", "e":
		if len(result.BannedList) > 0 && result.BannedCursorEdit < len(result.BannedList) {
			result.BannedEditing = true
			result.BannedInput.SetValue(result.BannedList[result.BannedCursorEdit])
			result.BannedInput.Focus()
			result.NeedsInputCmd = true
		}
	case "a":
		result.BannedList = append(result.BannedList, "new_function")
		result.BannedCursorEdit = len(result.BannedList) - 1
		result.BannedEditing = true
		result.BannedInput.SetValue("new_function")
		result.BannedInput.Focus()
		result.NeedsInputCmd = true
	case "d", "backspace":
		if len(result.BannedList) > 0 && result.BannedCursorEdit < len(result.BannedList) {
			result.BannedList = append(result.BannedList[:result.BannedCursorEdit], result.BannedList[result.BannedCursorEdit+1:]...)
			if result.BannedCursorEdit >= len(result.BannedList) && result.BannedCursorEdit > 0 {
				result.BannedCursorEdit--
			}
		}
	case "s", "ctrl+s":
		result.Save = true
	case "q", "esc":
		result.GoBack = true
		result.Save = true
	}
	return result
}
