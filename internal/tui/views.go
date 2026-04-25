package tui

import (
	"github.com/autoscan-lab/autoscan/internal/tui/components"
	"github.com/autoscan-lab/autoscan/internal/tui/views/banned"
	"github.com/autoscan-lab/autoscan/internal/tui/views/details"
	"github.com/autoscan-lab/autoscan/internal/tui/views/directory"
	"github.com/autoscan-lab/autoscan/internal/tui/views/home"
	policyview "github.com/autoscan-lab/autoscan/internal/tui/views/policy"
	"github.com/autoscan-lab/autoscan/internal/tui/views/settings"
	"github.com/autoscan-lab/autoscan/internal/tui/views/submissions"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	var content string

	switch m.currentView {
	case ViewHome:
		menuWidth := homeLayout(m.width)
		m.helpPanel.SetWidth(menuWidth)
		m.helpPanel.SetPolicyCount(len(m.policies))

		content = home.View(home.State{
			Width:         m.width,
			MenuWidth:     menuWidth,
			MenuItem:      int(m.menuItem),
			ConfirmDelete: m.confirmDelete,
			PolicyCount:   len(m.policies),
			AnimationView: m.eyeAnimation.View(),
			HelpPanelView: m.helpPanel.View(),
		})
	case ViewPolicySelect:
		content = policyview.SelectView(policyview.SelectState{
			Policies:       m.policies,
			SelectedPolicy: m.selectedPolicy,
			InputError:     m.inputError,
			Width:          m.width,
		})
	case ViewPolicyManage:
		content = policyview.ManageView(policyview.ManageState{
			Policies:           m.policies,
			PolicyManageCursor: m.policyManageCursor,
			ConfirmDelete:      m.confirmDelete,
			Width:              m.width,
		})
	case ViewPolicyEditor:
		// Only add help bar if NOT in a sub-mode (sub-modes render their own hints)
		if m.policyEditor.InSubMode() {
			content = m.policyEditor.View()
		} else {
			content = m.policyEditor.View() + "\n\n" + components.RenderHelpBar([]components.HelpItem{
				{Key: "tab", Desc: "next field"},
				{Key: "↑↓", Desc: "navigate"},
				{Key: "esc", Desc: "cancel"},
			})
		}
	case ViewBannedEditor:
		content = banned.View(banned.State{
			Width:            m.width,
			BannedList:       m.bannedList,
			BannedCursorEdit: m.bannedCursorEdit,
			BannedEditing:    m.bannedEditing,
			BannedInput:      m.bannedInput,
		})
	case ViewSettings:
		content = settings.View(settings.State{
			Settings:       &m.settings,
			SettingsCursor: m.settingsCursor,
			Width:          m.width,
		})
	case ViewDirectoryInput:
		content = directory.View(directory.State{
			Width:         m.width,
			InputError:    m.inputError,
			FolderBrowser: m.folderBrowser,
		})
	case ViewSubmissions:
		content = submissions.View(m.buildSubmissionsState())
	case ViewDetails:
		content = details.View(m.buildDetailsState())
	default:
		menuWidth := homeLayout(m.width)
		m.helpPanel.SetWidth(menuWidth)
		m.helpPanel.SetPolicyCount(len(m.policies))

		content = home.View(home.State{
			Width:         m.width,
			MenuWidth:     menuWidth,
			MenuItem:      int(m.menuItem),
			ConfirmDelete: m.confirmDelete,
			PolicyCount:   len(m.policies),
			AnimationView: m.eyeAnimation.View(),
			HelpPanelView: m.helpPanel.View(),
		})
	}

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Left,
		lipgloss.Top,
		content,
	)
}

func homeLayout(totalWidth int) (menuWidth int) {
	contentWidth := totalWidth - 4
	if contentWidth < 80 {
		contentWidth = 80
	}

	menuWidth = contentWidth * 70 / 100
	if menuWidth > 86 {
		menuWidth = 86
	}
	if menuWidth < 60 {
		menuWidth = 60
	}

	return menuWidth
}
