package policy

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felitrejos/autoscan-engine/pkg/policy"
	"github.com/felitrejos/autoscan/internal/tui/components"
)

type SelectState struct {
	Width          int
	Policies       []*policy.Policy
	SelectedPolicy int
	InputError     string
}

type SelectUpdateResult struct {
	SelectedPolicy int
	InputError     string
	GoBack         bool
	GoToDirectory  bool
}

func SelectView(s SelectState) string {
	var b strings.Builder

	b.WriteString(components.RenderHeader("Select a Policy"))

	boxWidth := components.BoxWidth(s.Width, 8, 60)

	if len(s.Policies) == 0 {
		box := components.WarningBoxStyle(boxWidth)
		content := components.WarningText.Render("No policies found!") + "\n\n" +
			components.SubtleText.Render("Create a policy via Manage Policies or edit ~/.config/autoscan/")
		b.WriteString(box.Render(content))
	} else {
		box := components.PrimaryBoxStyle().Width(boxWidth)

		var list strings.Builder

		list.WriteString(components.SubtleText.Render(fmt.Sprintf("Available policies: %d", len(s.Policies))))
		list.WriteString("\n\n")

		for i, p := range s.Policies {
			list.WriteString(components.RenderMenuItem(p.Name, i == s.SelectedPolicy))
			list.WriteString("\n")
		}

		b.WriteString(box.Render(list.String()))

		if s.SelectedPolicy < len(s.Policies) {
			b.WriteString("\n\n")

			detailBox := components.RoundedBox().Width(boxWidth)

			var details strings.Builder
			p := s.Policies[s.SelectedPolicy]

			details.WriteString(components.Highlight.Render("Policy Details"))
			details.WriteString("\n\n")

			details.WriteString(components.SubtleText.Render("  Name:     "))
			details.WriteString(p.Name)
			details.WriteString("\n")

			relPath, _ := filepath.Rel(".", p.FilePath)
			details.WriteString(components.SubtleText.Render("  File:     "))
			details.WriteString(filepath.Base(relPath))
			details.WriteString("\n")

			isMultiProcess := p.Run.MultiProcess != nil && p.Run.MultiProcess.Enabled
			details.WriteString(components.SubtleText.Render("  Mode:     "))
			if isMultiProcess {
				details.WriteString(components.SuccessText.Render("Multi-Process"))
			} else {
				details.WriteString("Single Process")
			}
			details.WriteString("\n")

			details.WriteString(components.SubtleText.Render("  Flags:    "))
			if len(p.Compile.Flags) > 0 {
				details.WriteString(strings.Join(p.Compile.Flags, " "))
			} else {
				details.WriteString(components.SubtleText.Render("(default)"))
			}
			details.WriteString("\n")

			if len(p.LibraryFiles) > 0 {
				details.WriteString(components.SubtleText.Render("  Libraries:"))
				details.WriteString(strings.Join(p.LibraryFiles, ", "))
				details.WriteString("\n")
			}

			details.WriteString("\n")

			if isMultiProcess {
				mp := p.Run.MultiProcess
				details.WriteString(components.PrimaryText.Render("  Executables"))
				details.WriteString("\n")
				for _, proc := range mp.Executables {
					details.WriteString(fmt.Sprintf("    • %s ", proc.Name))
					details.WriteString(components.SubtleText.Render(fmt.Sprintf("(%s)", proc.SourceFile)))
					if proc.StartDelayMs > 0 {
						details.WriteString(components.SubtleText.Render(fmt.Sprintf(" +%dms", proc.StartDelayMs)))
					}
					details.WriteString("\n")
				}

				if len(mp.TestScenarios) > 0 {
					details.WriteString("\n")
					details.WriteString(components.PrimaryText.Render(fmt.Sprintf("  Test Scenarios (%d)", len(mp.TestScenarios))))
					details.WriteString("\n")
					for i, scenario := range mp.TestScenarios {
						if i >= 3 {
							details.WriteString(components.SubtleText.Render(fmt.Sprintf("    ... and %d more", len(mp.TestScenarios)-3)))
							details.WriteString("\n")
							break
						}
						details.WriteString(fmt.Sprintf("    • %s\n", scenario.Name))
					}
				}
			} else {
				if p.Compile.SourceFile != "" {
					details.WriteString(components.SubtleText.Render("  Source:   "))
					details.WriteString(p.Compile.SourceFile)
					details.WriteString("\n")
				}

				if len(p.Run.TestCases) > 0 {
					details.WriteString(components.PrimaryText.Render(fmt.Sprintf("  Test Cases (%d)", len(p.Run.TestCases))))
					details.WriteString("\n")
					for i, tc := range p.Run.TestCases {
						if i >= 3 {
							details.WriteString(components.SubtleText.Render(fmt.Sprintf("    ... and %d more", len(p.Run.TestCases)-3)))
							details.WriteString("\n")
							break
						}
						details.WriteString(fmt.Sprintf("    • %s\n", tc.Name))
					}
				} else {
					details.WriteString(components.SubtleText.Render("  No test cases defined"))
					details.WriteString("\n")
				}
			}

			b.WriteString(detailBox.Render(details.String()))
		}
	}

	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "select"},
		{Key: "esc", Desc: "back"},
	}))

	if s.InputError != "" {
		b.WriteString("\n")
		b.WriteString(components.ErrorText.Render("  " + s.InputError))
	}

	return b.String()
}

func SelectUpdate(s SelectState, msg tea.KeyMsg) SelectUpdateResult {
	result := SelectUpdateResult{
		SelectedPolicy: s.SelectedPolicy,
		InputError:     s.InputError,
		GoBack:         false,
		GoToDirectory:  false,
	}

	switch msg.String() {
	case "j", "down":
		if result.SelectedPolicy < len(s.Policies)-1 {
			result.SelectedPolicy++
			result.InputError = ""
		}
	case "k", "up":
		if result.SelectedPolicy > 0 {
			result.SelectedPolicy--
			result.InputError = ""
		}
	case "enter":
		if len(s.Policies) > 0 {
			if result.SelectedPolicy < len(s.Policies) {
				p := s.Policies[result.SelectedPolicy]
				if p != nil {
					isMulti := p.Run.MultiProcess != nil && p.Run.MultiProcess.Enabled
					if isMulti {
						if len(p.Run.MultiProcess.Executables) == 0 {
							result.InputError = "Multi-process policy needs at least one executable"
							return result
						}
						for _, proc := range p.Run.MultiProcess.Executables {
							if strings.TrimSpace(proc.SourceFile) == "" {
								result.InputError = "Multi-process policy has a missing source file"
								return result
							}
						}
					} else if strings.TrimSpace(p.Compile.SourceFile) == "" {
						result.InputError = "Single-process policy requires source_file"
						return result
					}
				}
			}
			result.InputError = ""
			result.GoToDirectory = true
		}
	case "q", "esc":
		result.InputError = ""
		result.GoBack = true
	}

	return result
}
