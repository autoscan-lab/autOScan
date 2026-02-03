package details

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/feli05/autoscan/internal/domain"
	"github.com/feli05/autoscan/internal/tui/components"
)

func renderRunTab(s State) string {
	var b strings.Builder

	if !s.Result.Compile.OK {
		b.WriteString(components.ErrorText.Render("[!] Cannot run - compilation failed"))
		b.WriteString("\n\n")
		b.WriteString(components.SubtleText.Render("Fix compilation errors first."))
		return b.String()
	}

	if !s.KeepBinaries {
		b.WriteString(components.WarningText.Render("[!] Binaries not available"))
		b.WriteString("\n\n")
		b.WriteString(components.SubtleText.Render("Enable 'Keep Binaries' in Settings, then re-run."))
		return b.String()
	}

	if s.IsMultiProcess {
		return renderMultiProcessMode(s)
	}

	return renderSingleProcessMode(s)
}

func renderSingleProcessMode(s State) string {
	var b strings.Builder

	if s.IsExecuting {
		b.WriteString(s.SpinnerView)
		b.WriteString(" Running...")
		b.WriteString("\n\n")
		b.WriteString(components.WarningText.Render("Press Ctrl+K to cancel"))
		return b.String()
	}

	b.WriteString(components.Subtle.Render("Custom Execution"))
	b.WriteString("\n\n")

	argsLabel := "  Arguments: "
	if s.RunInputFocused == 0 {
		argsLabel = components.Highlight.Render("> ") + "Arguments: "
	}
	b.WriteString(argsLabel)
	b.WriteString(s.RunArgsInput.View())
	b.WriteString("\n")

	stdinLabel := "  Stdin:     "
	if s.RunInputFocused == 1 {
		stdinLabel = components.Highlight.Render("> ") + "Stdin:     "
	}
	b.WriteString(stdinLabel)
	b.WriteString(s.RunStdinInput.View())
	b.WriteString("\n\n")

	if s.RunInputFocused == 2 {
		b.WriteString(components.Highlight.Render("> "))
		b.WriteString(components.SelectedItem.Render("[ Run ]"))
	} else {
		b.WriteString("  ")
		b.WriteString(components.SubtleText.Render("[ Run ]"))
	}
	b.WriteString("\n")

	// Test cases from policy
	if len(s.TestCases) > 0 {
		b.WriteString("\n")
		b.WriteString(components.Subtle.Render("Preset Test Cases"))
		b.WriteString(components.SubtleText.Render(fmt.Sprintf(" (%d)", len(s.TestCases))))
		b.WriteString("\n\n")

		for i, tc := range s.TestCases {
			cursor := "  "
			style := components.NormalItem
			if s.RunInputFocused == 3+i {
				cursor = components.Highlight.Render("> ")
				style = components.SelectedItem
			}

			name := tc.Name
			if name == "" {
				name = fmt.Sprintf("Test %d", i+1)
			}

			argsInfo := ""
			if len(tc.Args) > 0 {
				argsInfo = fmt.Sprintf(" [%s]", strings.Join(tc.Args, " "))
			}

			b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, style.Render(name), components.SubtleText.Render(argsInfo)))
		}
	}

	// Last run result
	if s.RunResult != nil {
		b.WriteString("\n")
		b.WriteString(components.Subtle.Render("─── Last Result ───"))
		b.WriteString("\n\n")
		b.WriteString(renderExecuteResult(s))
	}

	// Test results
	if len(s.RunTestResults) > 0 {
		b.WriteString("\n")
		b.WriteString(components.Subtle.Render("─── Test Results ───"))
		b.WriteString("\n\n")

		passed := 0
		for _, tr := range s.RunTestResults {
			if tr.Passed {
				passed++
			}
		}

		if passed == len(s.RunTestResults) {
			b.WriteString(components.SuccessText.Render(fmt.Sprintf("All %d tests passed!", passed)))
		} else {
			b.WriteString(components.WarningText.Render(fmt.Sprintf("%d/%d tests passed", passed, len(s.RunTestResults))))
		}
		b.WriteString("\n\n")

		for _, tr := range s.RunTestResults {
			name := tr.TestCaseName
			if name == "" {
				name = "Test"
			}
			if tr.Passed {
				b.WriteString(components.SuccessText.Render(fmt.Sprintf("  [PASS] %s", name)))
			} else {
				b.WriteString(components.ErrorText.Render(fmt.Sprintf("  [FAIL] %s", name)))
			}
			b.WriteString(components.SubtleText.Render(fmt.Sprintf(" (exit %d, %dms)", tr.ExitCode, tr.Duration.Milliseconds())))
			b.WriteString("\n")
		}
	}

	return b.String()
}

func renderMultiProcessMode(s State) string {
	var b strings.Builder

	b.WriteString(components.Subtle.Render("Multi-Process Mode"))
	b.WriteString(components.SubtleText.Render(fmt.Sprintf(" (%d processes)", len(s.MultiProcessExecs))))
	b.WriteString("\n\n")

	for _, proc := range s.MultiProcessExecs {
		b.WriteString(fmt.Sprintf("  • %s (%s)", proc.Name, proc.SourceFile))
		if proc.StartDelayMs > 0 {
			b.WriteString(components.SubtleText.Render(fmt.Sprintf(" [delay: %dms]", proc.StartDelayMs)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if s.RunInputFocused == 0 {
		b.WriteString(components.Highlight.Render("> "))
		b.WriteString(components.SelectedItem.Render("[ Run ]"))
	} else {
		b.WriteString("  ")
		b.WriteString(components.NormalItem.Render("[ Run ]"))
	}
	b.WriteString("\n")

	// Test scenarios
	if len(s.TestScenarios) > 0 {
		b.WriteString("\n")
		b.WriteString(components.Subtle.Render("Test Scenarios"))
		b.WriteString(components.SubtleText.Render(fmt.Sprintf(" (%d)", len(s.TestScenarios))))
		b.WriteString("\n\n")

		for i, scenario := range s.TestScenarios {
			cursor := "  "
			style := components.NormalItem
			if s.RunInputFocused == 1+i {
				cursor = components.Highlight.Render("> ")
				style = components.SelectedItem
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(scenario.Name)))
		}
	}

	// Multi-process results
	if s.ShowMultiProcess && s.MultiProcessResult != nil {
		b.WriteString("\n")
		b.WriteString(renderMultiProcessGrid(s))
	}

	return b.String()
}

func renderMultiProcessGrid(s State) string {
	if s.MultiProcessResult == nil {
		return ""
	}

	var b strings.Builder

	if s.MultiProcessResult.ScenarioName != "" {
		b.WriteString(components.Subtle.Render(fmt.Sprintf("─── %s ───", s.MultiProcessResult.ScenarioName)))
	} else {
		b.WriteString(components.Subtle.Render("─── Multi-Process Results ───"))
	}
	b.WriteString("\n")
	b.WriteString(components.SubtleText.Render(fmt.Sprintf("Total: %dms", s.MultiProcessResult.TotalDuration.Milliseconds())))

	anyRunning := false
	anyKilled := false
	for _, pr := range s.MultiProcessResult.Processes {
		if pr.Running {
			anyRunning = true
		}
		if pr.Killed {
			anyKilled = true
		}
	}

	if anyRunning {
		b.WriteString(components.PrimaryText.Render(" [RUNNING...]"))
		b.WriteString(components.SubtleText.Render(" (Ctrl+K to kill)"))
	} else if s.MultiProcessResult.AllPassed {
		b.WriteString(components.SuccessText.Render(" [ALL PASSED]"))
	} else if anyKilled {
		b.WriteString(components.WarningText.Render(" [KILLED]"))
	} else if !s.MultiProcessResult.AllPassed {
		b.WriteString(components.WarningText.Render(" [Some failed]"))
	} else {
		b.WriteString(components.ErrorText.Render(" [Incomplete]"))
	}
	b.WriteString("\n\n")

	processes := s.MultiProcessResult.Order
	numProcs := len(processes)

	scenarioCount := len(s.TestScenarios)
	processStartIdx := 1 + scenarioCount

	availableWidth := s.Width - 14
	if availableWidth < 40 {
		availableWidth = 40
	}

	minColWidth := 38
	useTwoColumns := availableWidth >= (minColWidth*2 + 4)

	if useTwoColumns {
		colWidth := (availableWidth - 4) / 2

		for i := 0; i < numProcs; i += 2 {
			row := renderProcessRow(s, processes, i, colWidth, scenarioCount)
			b.WriteString(row)
			if i+2 < numProcs {
				b.WriteString("\n")
			}
		}
	} else {
		colWidth := availableWidth

		for i := 0; i < numProcs; i++ {
			procName := processes[i]
			proc := s.MultiProcessResult.Processes[procName]
			isSelected := s.RunInputFocused == processStartIdx+i
			isFocused := s.SelectedProcessIdx == i
			scrollOffset := 0
			if isFocused {
				scrollOffset = s.OutputScroll
			}
			b.WriteString(renderProcessBox(s, proc, colWidth, isSelected, isFocused, scrollOffset))
			if i < numProcs-1 {
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

func renderProcessRow(s State, processes []string, startIdx, colWidth, scenarioCount int) string {
	if startIdx >= len(processes) {
		return ""
	}

	processStartIdx := 1 + scenarioCount

	procName := processes[startIdx]
	proc := s.MultiProcessResult.Processes[procName]
	isSelected1 := s.RunInputFocused == processStartIdx+startIdx
	isFocused1 := s.SelectedProcessIdx == startIdx
	scroll1 := 0
	if isFocused1 {
		scroll1 = s.OutputScroll
	}
	box1 := renderProcessBox(s, proc, colWidth, isSelected1, isFocused1, scroll1)

	if startIdx+1 >= len(processes) {
		return box1
	}

	procName2 := processes[startIdx+1]
	proc2 := s.MultiProcessResult.Processes[procName2]
	isSelected2 := s.RunInputFocused == processStartIdx+startIdx+1
	isFocused2 := s.SelectedProcessIdx == startIdx+1
	scroll2 := 0
	if isFocused2 {
		scroll2 = s.OutputScroll
	}
	box2 := renderProcessBox(s, proc2, colWidth, isSelected2, isFocused2, scroll2)

	return lipgloss.JoinHorizontal(lipgloss.Top, box1, "  ", box2)
}

func renderProcessBox(s State, proc *domain.ProcessResult, width int, isSelected, isFocused bool, scrollOffset int) string {
	borderColor := components.Muted
	if isFocused {
		borderColor = components.Accent
	} else if isSelected {
		borderColor = components.PrimaryGlow
	} else if proc.Running {
		borderColor = components.Primary
	} else if proc.Killed {
		borderColor = components.Warning
	} else if proc.ExpectedExit != nil {
		if proc.Passed {
			borderColor = components.Success
		} else {
			borderColor = components.Error
		}
	}

	contentWidth := width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	var content strings.Builder

	allOutput := components.SanitizeDisplay(proc.Stdout)
	if proc.Stderr != "" {
		if allOutput != "" {
			allOutput += "\n" + components.WarningText.Render("stderr:") + "\n" + components.SanitizeDisplay(proc.Stderr)
		} else {
			allOutput = components.WarningText.Render("stderr:") + "\n" + components.SanitizeDisplay(proc.Stderr)
		}
	}

	wrappedLines := components.WrapLines(allOutput, contentWidth)
	maxShow := 8
	minContentLines := maxShow
	totalLines := len(wrappedLines)
	startIdx, endIdx := components.ScrollIndices(totalLines, maxShow, scrollOffset)

	header := proc.Name
	sourceInfo := fmt.Sprintf(" (%s)", proc.SourceFile)
	if len(header)+len(sourceInfo) > contentWidth-10 {
		maxSource := contentWidth - len(header) - 15
		if maxSource > 3 {
			sourceInfo = fmt.Sprintf(" (%s...)", proc.SourceFile[:maxSource])
		} else {
			sourceInfo = ""
		}
	}
	content.WriteString(components.Subtle.Render(header))
	content.WriteString(components.SubtleText.Render(sourceInfo))
	if isFocused && totalLines > maxShow {
		content.WriteString(components.SubtleText.Render(fmt.Sprintf(" [%d-%d/%d]", startIdx+1, endIdx, totalLines)))
	}
	content.WriteString("\n")

	content.WriteString(renderProcessStatusLine(proc))
	content.WriteString("\n")

	var outputLines []string
	if proc.OutputMatch == domain.OutputMatchFail && len(proc.OutputDiff) > 0 {
		outputLines = append(outputLines, renderDiffLines(proc.OutputDiff, contentWidth)...)
		if proc.Stderr != "" {
			outputLines = appendStderrBlock(outputLines, proc.Stderr, contentWidth)
		}
	} else {
		outputLines = wrappedLines
	}

	totalLines = len(outputLines)
	startIdx, endIdx = components.ScrollIndices(totalLines, maxShow, scrollOffset)

	for i := startIdx; i < endIdx; i++ {
		content.WriteString(outputLines[i])
		content.WriteString("\n")
	}

	currentLines := endIdx - startIdx
	for i := currentLines; i < minContentLines; i++ {
		content.WriteString("\n")
	}

	box := components.TableBoxStyle().
		BorderForeground(borderColor).
		Width(width)

	return box.Render(content.String())
}

func renderExecuteResult(s State) string {
	if s.RunResult == nil {
		return ""
	}

	r := *s.RunResult

	boxWidth := s.Width - 14
	if boxWidth < 40 {
		boxWidth = 40
	}
	contentWidth := boxWidth - 4

	testCaseCount := len(s.TestCases)
	outputBoxIdx := 3 + testCaseCount
	isSelected := s.RunInputFocused == outputBoxIdx
	isFocused := s.SelectedProcessIdx >= 0

	borderColor := components.Muted
	if isFocused {
		borderColor = components.Accent
	} else if isSelected {
		borderColor = components.PrimaryGlow
	} else if r.ExitCode == 0 && !r.TimedOut {
		borderColor = components.Success
	} else {
		borderColor = components.Warning
	}

	var content strings.Builder

	content.WriteString(renderExecuteStatusLine(r))

	var outputLines []string
	if r.OutputMatch == domain.OutputMatchFail && len(r.OutputDiff) > 0 {
		outputLines = append(outputLines, renderDiffLines(r.OutputDiff, contentWidth)...)
		if r.Stderr != "" {
			outputLines = appendStderrBlock(outputLines, r.Stderr, contentWidth)
		}
	} else {
		allOutput := components.SanitizeDisplay(r.Stdout)
		if r.Stderr != "" {
			if allOutput != "" {
				allOutput += "\n" + components.WarningText.Render("stderr:") + "\n" + components.SanitizeDisplay(r.Stderr)
			} else {
				allOutput = components.WarningText.Render("stderr:") + "\n" + components.SanitizeDisplay(r.Stderr)
			}
		}
		outputLines = components.WrapLines(allOutput, contentWidth)
	}

	maxShow := 15
	totalLines := len(outputLines)
	startIdx, endIdx := components.ScrollIndices(totalLines, maxShow, s.OutputScroll)

	if isFocused && totalLines > maxShow {
		content.WriteString(components.SubtleText.Render(fmt.Sprintf(" [%d-%d/%d]", startIdx+1, endIdx, totalLines)))
	}
	content.WriteString("\n")

	if totalLines > 0 {
		for i := startIdx; i < endIdx; i++ {
			content.WriteString(outputLines[i])
			content.WriteString("\n")
		}
		linesShown := endIdx - startIdx
		for i := linesShown; i < maxShow; i++ {
			content.WriteString("\n")
		}
	} else {
		content.WriteString(components.SubtleText.Render("(no output)\n"))
		for i := 1; i < maxShow; i++ {
			content.WriteString("\n")
		}
	}

	box := components.TableBoxStyle().
		BorderForeground(borderColor).
		Width(boxWidth)

	return box.Render(content.String())
}
