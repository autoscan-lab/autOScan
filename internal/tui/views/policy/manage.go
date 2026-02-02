package policy

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/feli05/autoscan/internal/policy"
	"github.com/feli05/autoscan/internal/tui/components"
	"github.com/feli05/autoscan/internal/tui/styles"
)

type ManageState struct {
	Width              int
	Policies           []*policy.Policy
	PolicyManageCursor int
	ConfirmDelete      bool
}

type ManageNavigation int

const (
	ManageNavNone ManageNavigation = iota
	ManageNavBack
	ManageNavBannedEditor
	ManageNavNewPolicy
	ManageNavEditPolicy
)

type ManageUpdateResult struct {
	PolicyManageCursor int
	ConfirmDelete      bool
	Navigation         ManageNavigation
	PolicyToEdit       *policy.Policy
	PolicyToDelete     *policy.Policy
}

func ManageView(s ManageState) string {
	var b strings.Builder

	b.WriteString(components.RenderHeader("Manage Policies"))

	boxWidth := components.BoxWidth(s.Width, 8, 60)

	configBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(1, 2).
		Width(boxWidth)

	var configSection strings.Builder
	configSection.WriteString(styles.SubtleText.Render("Configuration"))
	configSection.WriteString("\n\n")

	configSection.WriteString(components.RenderMenuItem("Edit Banned Functions", s.PolicyManageCursor == -1))
	configSection.WriteString("\n")
	configSection.WriteString(styles.SubtleText.Render("    Global list of prohibited function calls"))

	b.WriteString(configBox.Render(configSection.String()))
	b.WriteString("\n\n")

	policyBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2).
		Width(boxWidth)

	var policySection strings.Builder
	policySection.WriteString(styles.PrimaryText.Render(fmt.Sprintf("Policies (%d)", len(s.Policies))))
	policySection.WriteString("\n\n")

	policySection.WriteString(components.RenderMenuItem("+ Create New Policy", s.PolicyManageCursor == 0))
	policySection.WriteString("\n")

	if len(s.Policies) > 0 {
		policySection.WriteString("\n")
	}

	for i, p := range s.Policies {
		policySection.WriteString(components.RenderMenuItem(p.Name, s.PolicyManageCursor == i+1))
		policySection.WriteString("\n")
	}

	if s.PolicyManageCursor > 0 && s.PolicyManageCursor <= len(s.Policies) {
		p := s.Policies[s.PolicyManageCursor-1]
		policySection.WriteString("\n")
		policySection.WriteString(styles.SubtleText.Render(fmt.Sprintf("  File: %s", filepath.Base(p.FilePath))))
		if len(p.Compile.Flags) > 0 {
			policySection.WriteString("\n")
			policySection.WriteString(styles.SubtleText.Render(fmt.Sprintf("  Flags: %s", strings.Join(p.Compile.Flags, " "))))
		}
	}

	if s.ConfirmDelete && s.PolicyManageCursor > 0 {
		policySection.WriteString("\n")
		policySection.WriteString(components.ConfirmDialog("Delete this policy?"))
	}

	b.WriteString(policyBox.Render(policySection.String()))

	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "select"},
		{Key: "d", Desc: "delete"},
		{Key: "esc", Desc: "back"},
	}))

	return b.String()
}

func ManageUpdate(s ManageState, msg tea.KeyMsg) ManageUpdateResult {
	result := ManageUpdateResult{
		PolicyManageCursor: s.PolicyManageCursor,
		ConfirmDelete:      s.ConfirmDelete,
		Navigation:         ManageNavNone,
		PolicyToEdit:       nil,
		PolicyToDelete:     nil,
	}

	maxCursor := len(s.Policies)

	switch msg.String() {
	case "j", "down":
		if result.PolicyManageCursor < maxCursor {
			result.PolicyManageCursor++
		}
	case "k", "up":
		if result.PolicyManageCursor > -1 {
			result.PolicyManageCursor--
		}
	case "enter":
		if result.PolicyManageCursor == -1 {
			result.Navigation = ManageNavBannedEditor
		} else if result.PolicyManageCursor == 0 {
			result.Navigation = ManageNavNewPolicy
			result.PolicyToEdit = nil
		} else {
			result.Navigation = ManageNavEditPolicy
			result.PolicyToEdit = s.Policies[result.PolicyManageCursor-1]
		}
	case "e":
		if result.PolicyManageCursor > 0 && result.PolicyManageCursor <= len(s.Policies) {
			result.Navigation = ManageNavEditPolicy
			result.PolicyToEdit = s.Policies[result.PolicyManageCursor-1]
		}
	case "d":
		if result.PolicyManageCursor > 0 && result.PolicyManageCursor <= len(s.Policies) {
			result.ConfirmDelete = true
		}
	case "y":
		if result.ConfirmDelete && result.PolicyManageCursor > 0 && result.PolicyManageCursor <= len(s.Policies) {
			result.PolicyToDelete = s.Policies[result.PolicyManageCursor-1]
			result.ConfirmDelete = false
		}
	case "n":
		result.ConfirmDelete = false
	case "q", "esc":
		if result.ConfirmDelete {
			result.ConfirmDelete = false
		} else {
			result.Navigation = ManageNavBack
		}
	}

	return result
}
