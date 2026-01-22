package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/felipetrejos/autoscan/internal/config"
	"github.com/felipetrejos/autoscan/internal/policy"
	"github.com/felipetrejos/autoscan/internal/tui/styles"
	"gopkg.in/yaml.v3"
)

// PolicyEditorField represents which field is being edited
type PolicyEditorField int

const (
	FieldName PolicyEditorField = iota
	FieldFlags
	FieldOutput
	FieldRequiredFiles
	FieldSave
	FieldCancel
)

// PolicyEditor handles creating and editing policies
type PolicyEditor struct {
	isNew    bool
	filePath string

	// Input fields
	nameInput          textinput.Model
	flagsInput         textinput.Model
	outputInput        textinput.Model
	requiredFilesInput textinput.Model

	focusedField PolicyEditorField
	errorMsg     string
}

// NewPolicyEditor creates a new policy editor
func NewPolicyEditor(width, height int) PolicyEditor {
	nameInput := textinput.New()
	nameInput.Placeholder = "Lab 01 - Introduction"
	nameInput.CharLimit = 100
	nameInput.Width = 45
	nameInput.Focus()

	flagsInput := textinput.New()
	flagsInput.Placeholder = "-Wall -Wextra -lpthread"
	flagsInput.CharLimit = 200
	flagsInput.Width = 45
	flagsInput.SetValue("-Wall -Wextra")

	outputInput := textinput.New()
	outputInput.Placeholder = "lab01"
	outputInput.CharLimit = 50
	outputInput.Width = 45

	requiredFilesInput := textinput.New()
	requiredFilesInput.Placeholder = "S0.c S1.c (space-separated)"
	requiredFilesInput.CharLimit = 200
	requiredFilesInput.Width = 45

	return PolicyEditor{
		isNew:              true,
		nameInput:          nameInput,
		flagsInput:         flagsInput,
		outputInput:        outputInput,
		requiredFilesInput: requiredFilesInput,
		focusedField:       FieldName,
	}
}

// LoadPolicy loads an existing policy for editing
func (e *PolicyEditor) LoadPolicy(p *policy.Policy) {
	e.isNew = false
	e.filePath = p.FilePath

	e.nameInput.SetValue(p.Name)
	e.flagsInput.SetValue(strings.Join(p.Compile.Flags, " "))
	e.outputInput.SetValue(p.Compile.Output)
	e.requiredFilesInput.SetValue(strings.Join(p.RequiredFiles, " "))
}

// Reset resets the editor for a new policy
func (e *PolicyEditor) Reset() {
	e.isNew = true
	e.filePath = ""
	e.focusedField = FieldName
	e.errorMsg = ""

	e.nameInput.SetValue("")
	e.nameInput.Focus()
	e.flagsInput.SetValue("-Wall -Wextra")
	e.outputInput.SetValue("")
	e.requiredFilesInput.SetValue("")
}

// Update handles input for the policy editor
func (e *PolicyEditor) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			e.nextField()
			return nil
		case "shift+tab", "up":
			e.prevField()
			return nil
		case "enter":
			if e.focusedField == FieldSave {
				return e.save()
			}
			e.nextField()
			return nil
		}
	}

	// Update the focused input
	var cmd tea.Cmd
	switch e.focusedField {
	case FieldName:
		e.nameInput, cmd = e.nameInput.Update(msg)
	case FieldFlags:
		e.flagsInput, cmd = e.flagsInput.Update(msg)
	case FieldOutput:
		e.outputInput, cmd = e.outputInput.Update(msg)
	case FieldRequiredFiles:
		e.requiredFilesInput, cmd = e.requiredFilesInput.Update(msg)
	}

	return cmd
}

func (e *PolicyEditor) nextField() {
	e.blurAll()
	e.focusedField++
	if e.focusedField > FieldCancel {
		e.focusedField = FieldName
	}
	e.focusCurrent()
}

func (e *PolicyEditor) prevField() {
	e.blurAll()
	if e.focusedField == FieldName {
		e.focusedField = FieldCancel
	} else {
		e.focusedField--
	}
	e.focusCurrent()
}

func (e *PolicyEditor) blurAll() {
	e.nameInput.Blur()
	e.flagsInput.Blur()
	e.outputInput.Blur()
	e.requiredFilesInput.Blur()
}

func (e *PolicyEditor) focusCurrent() {
	switch e.focusedField {
	case FieldName:
		e.nameInput.Focus()
	case FieldFlags:
		e.flagsInput.Focus()
	case FieldOutput:
		e.outputInput.Focus()
	case FieldRequiredFiles:
		e.requiredFilesInput.Focus()
	}
}

func (e *PolicyEditor) save() tea.Cmd {
	return func() tea.Msg {
		// Validate
		name := strings.TrimSpace(e.nameInput.Value())
		if name == "" {
			return policySaveErrorMsg{err: "Policy name is required"}
		}

		output := strings.TrimSpace(e.outputInput.Value())
		if output == "" {
			output = "a.out"
		}

		// Parse flags
		flagsStr := strings.TrimSpace(e.flagsInput.Value())
		var flags []string
		if flagsStr != "" {
			flags = strings.Fields(flagsStr)
		}

		// Parse required files
		reqStr := strings.TrimSpace(e.requiredFilesInput.Value())
		var requiredFiles []string
		if reqStr != "" {
			requiredFiles = strings.Fields(reqStr)
		}

		// Build policy struct for YAML
		p := struct {
			Name     string `yaml:"name"`
			Root     string `yaml:"root"`
			Discover struct {
				LeafSubmission bool `yaml:"leaf_submission"`
				MinCFiles      int  `yaml:"min_c_files"`
			} `yaml:"discover"`
			Compile struct {
				GCC    string   `yaml:"gcc"`
				Flags  []string `yaml:"flags"`
				Output string   `yaml:"output"`
			} `yaml:"compile"`
			RequiredFiles []string `yaml:"required_files,omitempty"`
			Report        struct {
				Export struct {
					Markdown bool `yaml:"markdown"`
				} `yaml:"export"`
			} `yaml:"report"`
		}{}

		p.Name = name
		p.Root = "."
		p.Discover.LeafSubmission = true
		p.Discover.MinCFiles = 1
		p.Compile.GCC = "gcc"
		p.Compile.Flags = flags
		p.Compile.Output = output
		p.RequiredFiles = requiredFiles
		p.Report.Export.Markdown = true

		// Marshal to YAML
		data, err := yaml.Marshal(p)
		if err != nil {
			return policySaveErrorMsg{err: fmt.Sprintf("Failed to create YAML: %v", err)}
		}

		// Determine file path
		var filePath string
		if e.isNew {
			// Generate filename from name
			safeName := strings.ToLower(name)
			safeName = strings.ReplaceAll(safeName, " ", "-")
			safeName = strings.ReplaceAll(safeName, "/", "-")
			safeName = strings.ReplaceAll(safeName, "\\", "-")

			// Get policies directory from config
			policiesDir, err := config.PoliciesDir()
			if err != nil {
				return policySaveErrorMsg{err: fmt.Sprintf("Failed to get config dir: %v", err)}
			}

			if err := os.MkdirAll(policiesDir, 0755); err != nil {
				return policySaveErrorMsg{err: fmt.Sprintf("Failed to create directory: %v", err)}
			}

			filePath = filepath.Join(policiesDir, safeName+".yaml")
		} else {
			filePath = e.filePath
		}

		// Write file
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return policySaveErrorMsg{err: fmt.Sprintf("Failed to save: %v", err)}
		}

		return policySavedMsg{path: filePath, isNew: e.isNew}
	}
}

// View renders the policy editor
func (e *PolicyEditor) View() string {
	var b strings.Builder

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		Padding(1, 2)

	title := "Edit Policy"
	if e.isNew {
		title = "Create New Policy"
	}
	b.WriteString(header.Render(title))
	b.WriteString("\n\n")

	// Form box
	formBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(1, 2).
		Width(60)

	var form strings.Builder

	// Name field
	form.WriteString(e.renderField("Policy Name", e.nameInput.View(), FieldName))
	form.WriteString("\n")

	// Flags field
	form.WriteString(e.renderField("Compiler Flags", e.flagsInput.View(), FieldFlags))
	form.WriteString(styles.Subtle.Render("  (e.g., -Wall -Wextra -lpthread -lm)\n"))
	form.WriteString("\n")

	// Output field
	form.WriteString(e.renderField("Output Binary", e.outputInput.View(), FieldOutput))
	form.WriteString("\n")

	// Required files field
	form.WriteString(e.renderField("Required Source Files", e.requiredFilesInput.View(), FieldRequiredFiles))
	form.WriteString(styles.Subtle.Render("  (e.g., S0.c S1.c divide.c)\n"))

	b.WriteString(formBox.Render(form.String()))
	b.WriteString("\n\n")

	// Buttons
	var buttons strings.Builder
	buttons.WriteString("  ")

	saveStyle := styles.NormalItem
	if e.focusedField == FieldSave {
		saveStyle = styles.SelectedItem
	}
	buttons.WriteString(saveStyle.Render(" Save "))
	buttons.WriteString("  ")

	cancelStyle := styles.NormalItem
	if e.focusedField == FieldCancel {
		cancelStyle = styles.SelectedItem
	}
	buttons.WriteString(cancelStyle.Render(" Cancel "))

	b.WriteString(buttons.String())

	// Error message
	if e.errorMsg != "" {
		b.WriteString("\n\n")
		b.WriteString(styles.ErrorStyle.Render("  Error: " + e.errorMsg))
	}

	// Help bar
	b.WriteString("\n\n")
	helpStyle := lipgloss.NewStyle().
		Foreground(styles.Muted).
		Background(lipgloss.Color("#1F2937")).
		Padding(0, 1)
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)

	var help strings.Builder
	help.WriteString(keyStyle.Render("tab") + helpStyle.Render(" next  "))
	help.WriteString(keyStyle.Render("enter") + helpStyle.Render(" save  "))
	help.WriteString(keyStyle.Render("esc") + helpStyle.Render(" cancel"))
	b.WriteString(helpStyle.Render(help.String()))

	return b.String()
}

func (e *PolicyEditor) renderField(label, input string, field PolicyEditorField) string {
	var b strings.Builder

	if e.focusedField == field {
		b.WriteString(styles.Highlight.Render("> " + label))
	} else {
		b.WriteString(styles.Subtle.Render("  " + label))
	}
	b.WriteString("\n")
	b.WriteString("  " + input + "\n")

	return b.String()
}

// Message types for policy editor
type (
	policySavedMsg struct {
		path  string
		isNew bool
	}
	policySaveErrorMsg struct {
		err string
	}
	policyDeletedMsg struct {
		name string
	}
)

// DeletePolicy deletes a policy file
func DeletePolicy(p *policy.Policy) tea.Cmd {
	return func() tea.Msg {
		if err := os.Remove(p.FilePath); err != nil {
			return errorMsg(err)
		}
		return policyDeletedMsg{name: p.Name}
	}
}
