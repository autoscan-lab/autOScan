package details

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/autoscan-lab/autoscan-engine/pkg/domain"
	"github.com/autoscan-lab/autoscan-engine/pkg/policy"
	"github.com/autoscan-lab/autoscan/internal/tui/components"
)

type State struct {
	Width  int
	Height int

	// Current submission result
	Result       domain.SubmissionResult
	SubmissionID string

	// Tab navigation
	DetailsTab   int // 0=Compile, 1=Banned, 2=Files, 3=Run
	DetailScroll int

	// Banned tab state
	BannedCursor  int
	ExpandedFuncs map[string]bool

	// Run tab state
	RunInputFocused    int
	SelectedProcessIdx int
	OutputScroll       int
	IsExecuting        bool
	SpinnerView        string

	// Text inputs for single-process mode
	RunArgsInput  textinput.Model
	RunStdinInput textinput.Model

	// Execution results
	RunResult          *domain.ExecuteResult
	RunTestResults     []domain.ExecuteResult
	MultiProcessResult *domain.MultiProcessResult
	ShowMultiProcess   bool

	// Policy context
	IsMultiProcess    bool
	TestCases         []policy.TestCase
	TestScenarios     []policy.MultiProcessScenario
	MultiProcessExecs []policy.ProcessConfig
	KeepBinaries      bool
}

type UpdateResult struct {
	DetailsTab   int
	DetailScroll int

	BannedCursor  int
	ExpandedFuncs map[string]bool

	RunInputFocused    int
	SelectedProcessIdx int
	OutputScroll       int
	ShowMultiProcess   bool

	RunArgsInput  textinput.Model
	RunStdinInput textinput.Model

	// Navigation
	GoBack bool

	// Execution commands (handlers.go interprets these)
	ExecuteSubmission   bool
	ExecuteTestCase     int // -1 = none, 0+ = test case index
	ExecuteMultiProcess bool
	ExecuteScenario     int // -1 = none, 0+ = scenario index
	CancelExecution     bool

	Cmd tea.Cmd
}

func View(s State) string {
	var b strings.Builder

	// Header with submission ID
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(components.Primary).
		Padding(1, 2)
	b.WriteString(header.Render(s.SubmissionID))
	b.WriteString("\n")

	// Tab bar
	tabs := []string{"Compile", "Banned", "Files", "Run"}
	var tabRow strings.Builder
	tabRow.WriteString("  ")
	for i, tab := range tabs {
		if i == s.DetailsTab {
			tabRow.WriteString(components.TabActive.Render(fmt.Sprintf(" %s ", tab)))
		} else {
			tabRow.WriteString(components.TabInactive.Render(fmt.Sprintf(" %s ", tab)))
		}
		tabRow.WriteString(" ")
	}
	b.WriteString(tabRow.String())
	b.WriteString("\n\n")

	// Content box
	contentWidth := s.Width - 8
	if contentWidth < 80 {
		contentWidth = 80
	}
	contentBox := components.RoundedBox().Width(contentWidth)

	var content string
	switch s.DetailsTab {
	case 0:
		content = renderCompileTab(s)
	case 1:
		content = renderBannedTab(s)
	case 2:
		content = renderFilesTab(s)
	case 3:
		content = renderRunTab(s)
	}

	b.WriteString(contentBox.Render(content))
	b.WriteString("\n\n")

	// Help bar
	switch s.DetailsTab {
	case 1:
		b.WriteString(components.RenderHelpBar([]components.HelpItem{
			{Key: "tab", Desc: "switch tabs"},
			{Key: "↑/↓", Desc: "navigate"},
			{Key: "enter", Desc: "expand/collapse"},
			{Key: "esc", Desc: "back"},
		}))
	case 3:
		helpItems := []components.HelpItem{
			{Key: "tab", Desc: "switch tabs"},
			{Key: "↑/↓", Desc: "navigate"},
			{Key: "enter", Desc: "run/focus"},
		}
		if s.IsMultiProcess {
			helpItems = append(helpItems, components.HelpItem{Key: "m", Desc: "multi-process"})
		}
		helpItems = append(helpItems, components.HelpItem{Key: "esc", Desc: "back"})
		b.WriteString(components.RenderHelpBar(helpItems))
	default:
		b.WriteString(components.RenderHelpBar([]components.HelpItem{
			{Key: "tab", Desc: "switch tabs"},
			{Key: "↑/↓", Desc: "scroll"},
			{Key: "esc", Desc: "back"},
		}))
	}

	return b.String()
}

func Update(s State, msg tea.KeyMsg) UpdateResult {
	result := UpdateResult{
		DetailsTab:         s.DetailsTab,
		DetailScroll:       s.DetailScroll,
		BannedCursor:       s.BannedCursor,
		ExpandedFuncs:      s.ExpandedFuncs,
		RunInputFocused:    s.RunInputFocused,
		SelectedProcessIdx: s.SelectedProcessIdx,
		OutputScroll:       s.OutputScroll,
		ShowMultiProcess:   s.ShowMultiProcess,
		RunArgsInput:       s.RunArgsInput,
		RunStdinInput:      s.RunStdinInput,
		ExecuteTestCase:    -1,
		ExecuteScenario:    -1,
	}

	// Run tab has its own update logic
	if s.DetailsTab == 3 {
		return updateRunTab(s, msg, result)
	}

	switch msg.String() {
	case "tab":
		result.DetailsTab = (s.DetailsTab + 1) % 4
		result.DetailScroll = 0
		result.BannedCursor = 0
		if result.DetailsTab == 3 {
			result.RunInputFocused = 0
			result.RunArgsInput.Focus()
			result.RunStdinInput.Blur()
		}
	case "shift+tab":
		result.DetailsTab = (s.DetailsTab + 3) % 4
		result.DetailScroll = 0
		result.BannedCursor = 0
		if result.DetailsTab == 3 {
			result.RunInputFocused = 0
			result.RunArgsInput.Focus()
			result.RunStdinInput.Blur()
		}
	case "j", "down":
		if s.DetailsTab == 1 {
			maxCursor := len(s.Result.Scan.HitsByFunction) - 1
			if maxCursor >= 0 && s.BannedCursor < maxCursor {
				result.BannedCursor++
			}
		} else {
			result.DetailScroll++
		}
	case "k", "up":
		if s.DetailsTab == 1 {
			if s.BannedCursor > 0 {
				result.BannedCursor--
			}
		} else if s.DetailScroll > 0 {
			result.DetailScroll--
		}
	case "enter", " ":
		if s.DetailsTab == 1 {
			if result.ExpandedFuncs == nil {
				result.ExpandedFuncs = make(map[string]bool)
			}
			funcNames := getBannedFuncNames(s)
			if s.BannedCursor < len(funcNames) {
				fn := funcNames[s.BannedCursor]
				result.ExpandedFuncs[fn] = !result.ExpandedFuncs[fn]
			}
		}
	case "q", "esc":
		result.GoBack = true
		result.ExpandedFuncs = nil
		result.BannedCursor = 0
	}

	return result
}

func updateRunTab(s State, msg tea.KeyMsg, result UpdateResult) UpdateResult {
	// Handle execution cancellation
	if s.IsExecuting {
		switch msg.String() {
		case "ctrl+k", "K":
			result.CancelExecution = true
			return result
		}
		return result
	}

	if s.IsMultiProcess {
		return updateMultiProcessMode(s, msg, result)
	}

	return updateSingleProcessMode(s, msg, result)
}

func updateSingleProcessMode(s State, msg tea.KeyMsg, result UpdateResult) UpdateResult {
	testCaseCount := len(s.TestCases)
	maxFocus := 2 + testCaseCount
	outputBoxIdx := maxFocus + 1

	// If focused on output box, handle scrolling
	if s.RunResult != nil && s.SelectedProcessIdx >= 0 {
		maxScroll := calculateOutputMaxScroll(s)

		switch msg.String() {
		case "up", "k":
			if s.OutputScroll > 0 {
				result.OutputScroll--
			}
			return result
		case "down", "j":
			if s.OutputScroll < maxScroll {
				result.OutputScroll++
			}
			return result
		case "esc", "enter":
			result.SelectedProcessIdx = -1
			result.OutputScroll = 0
			result.RunInputFocused = outputBoxIdx
			return result
		}
		return result
	}

	switch msg.String() {
	case "tab":
		result.DetailsTab = 0
		result.DetailScroll = 0
		result.RunArgsInput.Blur()
		result.RunStdinInput.Blur()
		return result

	case "shift+tab":
		result.DetailsTab = 2
		result.DetailScroll = 0
		result.RunArgsInput.Blur()
		result.RunStdinInput.Blur()
		return result

	case "down", "j":
		maxIdx := maxFocus
		if s.RunResult != nil {
			maxIdx = outputBoxIdx
		}
		if s.RunInputFocused < maxIdx {
			result.RunInputFocused++
		}
		result.RunArgsInput.Blur()
		result.RunStdinInput.Blur()
		if result.RunInputFocused == 0 {
			result.RunArgsInput.Focus()
		} else if result.RunInputFocused == 1 {
			result.RunStdinInput.Focus()
		}
		return result

	case "up", "k":
		if s.RunInputFocused > 0 {
			result.RunInputFocused--
		}
		result.RunArgsInput.Blur()
		result.RunStdinInput.Blur()
		if result.RunInputFocused == 0 {
			result.RunArgsInput.Focus()
		} else if result.RunInputFocused == 1 {
			result.RunStdinInput.Focus()
		}
		return result

	case "enter":
		if s.RunResult != nil && s.RunInputFocused == outputBoxIdx {
			result.SelectedProcessIdx = 0
			result.OutputScroll = 0
			return result
		}
		if s.RunInputFocused == 2 {
			result.ExecuteSubmission = true
			return result
		} else if s.RunInputFocused > 2 && s.RunInputFocused <= maxFocus {
			result.ExecuteTestCase = s.RunInputFocused - 3
			return result
		}

	case "r":
		if s.RunInputFocused >= 2 {
			result.ExecuteSubmission = true
			return result
		}

	case "q", "esc":
		result.GoBack = true
		result.ExpandedFuncs = nil
		return result

	default:
		// Pass to text inputs if focused
		if s.RunInputFocused == 0 {
			var cmd tea.Cmd
			result.RunArgsInput, cmd = s.RunArgsInput.Update(msg)
			result.Cmd = cmd
		} else if s.RunInputFocused == 1 {
			var cmd tea.Cmd
			result.RunStdinInput, cmd = s.RunStdinInput.Update(msg)
			result.Cmd = cmd
		}
	}

	return result
}

func updateMultiProcessMode(s State, msg tea.KeyMsg, result UpdateResult) UpdateResult {
	scenarioCount := len(s.TestScenarios)
	maxFocus := scenarioCount

	// If focused on a process output, handle scrolling
	if s.MultiProcessResult != nil && s.SelectedProcessIdx >= 0 {
		maxScroll := calculateProcessMaxScroll(s)

		switch msg.String() {
		case "up", "k":
			if s.OutputScroll > 0 {
				result.OutputScroll--
			}
			return result
		case "down", "j":
			if s.OutputScroll < maxScroll {
				result.OutputScroll++
			}
			return result
		case "esc", "enter":
			result.SelectedProcessIdx = -1
			result.OutputScroll = 0
			return result
		}
		return result
	}

	// If we have multi-process results displayed
	if s.MultiProcessResult != nil && len(s.MultiProcessResult.Order) > 0 {
		numProcs := len(s.MultiProcessResult.Order)
		processStartIdx := 1 + scenarioCount

		switch msg.String() {
		case "tab":
			result.DetailsTab = 0
			result.DetailScroll = 0
			return result

		case "shift+tab":
			result.DetailsTab = 2
			result.DetailScroll = 0
			return result

		case "down", "j":
			maxIdx := processStartIdx + numProcs - 1
			if s.RunInputFocused < maxIdx {
				result.RunInputFocused++
			}
			return result

		case "up", "k":
			if s.RunInputFocused > 0 {
				result.RunInputFocused--
			}
			return result

		case "enter":
			if s.RunInputFocused == 0 {
				result.ExecuteMultiProcess = true
				return result
			} else if s.RunInputFocused > 0 && s.RunInputFocused <= scenarioCount {
				result.ExecuteScenario = s.RunInputFocused - 1
				return result
			} else if s.RunInputFocused >= processStartIdx {
				result.SelectedProcessIdx = s.RunInputFocused - processStartIdx
				result.OutputScroll = 0
				return result
			}

		case "m":
			result.ExecuteMultiProcess = true
			return result

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(msg.String()[0] - '1')
			if idx >= 0 && idx < scenarioCount {
				result.ExecuteScenario = idx
				return result
			}

		case "esc", "q":
			result.GoBack = true
			result.ExpandedFuncs = nil
			result.ShowMultiProcess = false
			return result
		}
		return result
	}

	// No results yet, basic navigation
	switch msg.String() {
	case "tab":
		result.DetailsTab = 0
		result.DetailScroll = 0
		return result

	case "shift+tab":
		result.DetailsTab = 2
		result.DetailScroll = 0
		return result

	case "down", "j":
		if s.RunInputFocused < maxFocus {
			result.RunInputFocused++
		}
		return result

	case "up", "k":
		if s.RunInputFocused > 0 {
			result.RunInputFocused--
		}
		return result

	case "enter":
		if s.RunInputFocused == 0 {
			result.ExecuteMultiProcess = true
			return result
		} else if s.RunInputFocused > 0 && s.RunInputFocused <= scenarioCount {
			result.ExecuteScenario = s.RunInputFocused - 1
			return result
		}

	case "m":
		result.ExecuteMultiProcess = true
		return result

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0] - '1')
		if idx >= 0 && idx < scenarioCount {
			result.ExecuteScenario = idx
			return result
		}

	case "esc", "q":
		result.GoBack = true
		result.ExpandedFuncs = nil
		result.ShowMultiProcess = false
		return result
	}

	return result
}

func calculateOutputMaxScroll(s State) int {
	if s.RunResult == nil {
		return 0
	}

	boxWidth := s.Width - 14
	if boxWidth < 40 {
		boxWidth = 40
	}
	contentWidth := boxWidth - 4

	var outputLen int
	if s.RunResult.OutputMatch == domain.OutputMatchFail && len(s.RunResult.OutputDiff) > 0 {
		outputLen = len(s.RunResult.OutputDiff)
		if s.RunResult.Stderr != "" {
			outputLen += 2 + len(components.WrapLines(s.RunResult.Stderr, contentWidth))
		}
	} else {
		allOutput := s.RunResult.Stdout
		if s.RunResult.Stderr != "" {
			if allOutput != "" {
				allOutput += "\nstderr:\n" + s.RunResult.Stderr
			} else {
				allOutput = "stderr:\n" + s.RunResult.Stderr
			}
		}
		outputLen = len(components.WrapLines(allOutput, contentWidth))
	}

	maxScroll := outputLen - 15
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

func calculateProcessMaxScroll(s State) int {
	if s.MultiProcessResult == nil || s.SelectedProcessIdx < 0 {
		return 0
	}

	numProcs := len(s.MultiProcessResult.Order)
	if s.SelectedProcessIdx >= numProcs {
		return 0
	}

	procName := s.MultiProcessResult.Order[s.SelectedProcessIdx]
	proc := s.MultiProcessResult.Processes[procName]

	boxWidth := (s.Width - 20) / 2
	if boxWidth < 30 {
		boxWidth = 30
	}
	contentWidth := boxWidth - 4

	var outputLen int
	if proc.OutputMatch == domain.OutputMatchFail && len(proc.OutputDiff) > 0 {
		outputLen = len(proc.OutputDiff)
		if proc.Stderr != "" {
			outputLen += 2 + len(components.WrapLines(proc.Stderr, contentWidth))
		}
	} else {
		allOutput := proc.Stdout
		if proc.Stderr != "" {
			if allOutput != "" {
				allOutput += "\nstderr:\n" + proc.Stderr
			} else {
				allOutput = "stderr:\n" + proc.Stderr
			}
		}
		outputLen = len(components.WrapLines(allOutput, contentWidth))
	}

	maxScroll := outputLen - 8
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}
