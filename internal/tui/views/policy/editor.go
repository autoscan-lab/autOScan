package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/felitrejos/autoscan/internal/config"
	"github.com/felitrejos/autoscan/internal/policy"
	"github.com/felitrejos/autoscan/internal/tui/components"
	"gopkg.in/yaml.v3"
)

type EditorField int

const (
	FieldName EditorField = iota
	FieldFlags
	FieldLibraryFiles
	FieldTestFiles
	FieldMultiProcessToggle
	FieldSourceFile
	FieldTestCases
	FieldMultiProcess
	FieldMultiProcessTests
	FieldSave
	FieldCancel
)

type DeleteErrorMsg struct {
	Err error
}

type Editor struct {
	isNew    bool
	filePath string
	width    int

	nameInput       textinput.Model
	flagsInput      textinput.Model
	sourceFileInput textinput.Model

	libraryFiles       []string
	libraryFilesCursor int
	testFiles          []string
	testFilesCursor    int

	folderBrowser    components.FolderBrowser
	browsingForLibs  bool
	browsingForTests bool
	browsingStartDir string

	showingExistingLibs bool
	existingLibs        []string
	existingLibsCursor  int

	showingExistingTests bool
	existingTests        []string
	existingTestsCursor  int

	// Expected output files for test cases (single-process)
	browsingForExpectedOutput     bool
	showingExistingExpectedOutput bool
	existingExpectedOutputs       []string
	existingExpectedOutputsCursor int

	// Expected output files for scenarios (multi-process)
	browsingForScenarioExpectedOutput     bool
	showingExistingScenarioExpectedOutput bool
	scenarioExpectedOutputProcess         string // which process we're selecting for

	testCases          []policy.TestCase
	testCasesCursor    int
	editingTestCase    bool
	editingTestCaseIdx int
	testCaseInputs     struct {
		name               textinput.Model
		args               textinput.Model
		input              textinput.Model
		expectedExit       textinput.Model
		expectedOutputFile string
		focusedInput       int
	}

	multiProcessEnabled bool
	multiProcessExecs   []policy.ProcessConfig
	multiProcessCursor  int
	editingProcess      bool
	editingProcessIdx   int
	processInputs       struct {
		name       textinput.Model
		sourceFile textinput.Model
		args       textinput.Model
		delayMs    textinput.Model
		focusedIdx int
	}

	testScenarios       []policy.MultiProcessScenario
	testScenariosCursor int
	editingScenario     bool
	editingScenarioIdx  int
	scenarioInputs      struct {
		name            textinput.Model
		processArgs     map[string]textinput.Model
		processStdin    map[string]textinput.Model
		processExit     map[string]textinput.Model
		expectedOutputs map[string]string // file paths per process
		focusedIdx      int
	}

	focusedField EditorField
	ErrorMsg     string
}

func NewEditor(width, height int) Editor {
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

	sourceFileInput := textinput.New()
	sourceFileInput.Placeholder = "S5.c (binary will be S5)"
	sourceFileInput.CharLimit = 50
	sourceFileInput.Width = 45

	cwd, _ := os.Getwd()

	tcNameInput := textinput.New()
	tcNameInput.Placeholder = "Test name"
	tcNameInput.CharLimit = 50
	tcNameInput.Width = 40

	tcArgsInput := textinput.New()
	tcArgsInput.Placeholder = "arg1 arg2 arg3"
	tcArgsInput.CharLimit = 200
	tcArgsInput.Width = 40

	tcInputInput := textinput.New()
	tcInputInput.Placeholder = "stdin (use \\n for newlines)"
	tcInputInput.CharLimit = 500
	tcInputInput.Width = 40

	tcExitInput := textinput.New()
	tcExitInput.Placeholder = "0"
	tcExitInput.CharLimit = 5
	tcExitInput.Width = 10

	procNameInput := textinput.New()
	procNameInput.Placeholder = "Producer"
	procNameInput.CharLimit = 30
	procNameInput.Width = 30

	procSourceInput := textinput.New()
	procSourceInput.Placeholder = "producer.c"
	procSourceInput.CharLimit = 50
	procSourceInput.Width = 30

	procArgsInput := textinput.New()
	procArgsInput.Placeholder = "arg1 arg2"
	procArgsInput.CharLimit = 100
	procArgsInput.Width = 30

	procDelayInput := textinput.New()
	procDelayInput.Placeholder = "0"
	procDelayInput.CharLimit = 10
	procDelayInput.Width = 10

	scenarioNameInput := textinput.New()
	scenarioNameInput.Placeholder = "Test Scenario Name"
	scenarioNameInput.CharLimit = 50
	scenarioNameInput.Width = 40

	pe := Editor{
		isNew:             true,
		nameInput:         nameInput,
		flagsInput:        flagsInput,
		sourceFileInput:   sourceFileInput,
		libraryFiles:      []string{},
		folderBrowser:     components.NewFolderBrowser(cwd),
		browsingStartDir:  cwd,
		focusedField:      FieldName,
		testCases:         []policy.TestCase{},
		multiProcessExecs: []policy.ProcessConfig{},
		testScenarios:     []policy.MultiProcessScenario{},
	}

	pe.testCaseInputs.name = tcNameInput
	pe.testCaseInputs.args = tcArgsInput
	pe.testCaseInputs.input = tcInputInput
	pe.testCaseInputs.expectedExit = tcExitInput

	pe.processInputs.name = procNameInput
	pe.processInputs.sourceFile = procSourceInput
	pe.processInputs.args = procArgsInput
	pe.processInputs.delayMs = procDelayInput

	pe.scenarioInputs.name = scenarioNameInput
	pe.scenarioInputs.processArgs = make(map[string]textinput.Model)
	pe.scenarioInputs.processStdin = make(map[string]textinput.Model)
	pe.scenarioInputs.processExit = make(map[string]textinput.Model)
	pe.scenarioInputs.expectedOutputs = make(map[string]string)

	return pe
}

func (e *Editor) LoadPolicy(p *policy.Policy) {
	e.isNew = false
	e.filePath = p.FilePath

	e.nameInput.SetValue(p.Name)
	e.flagsInput.SetValue(strings.Join(p.Compile.Flags, " "))
	e.sourceFileInput.SetValue(p.Compile.SourceFile)

	e.libraryFiles = make([]string, len(p.LibraryFiles))
	copy(e.libraryFiles, p.LibraryFiles)
	e.libraryFilesCursor = 0

	e.testFiles = make([]string, len(p.TestFiles))
	copy(e.testFiles, p.TestFiles)
	e.testFilesCursor = 0

	e.testCases = make([]policy.TestCase, len(p.Run.TestCases))
	copy(e.testCases, p.Run.TestCases)
	e.testCasesCursor = 0

	if p.Run.MultiProcess != nil {
		e.multiProcessEnabled = p.Run.MultiProcess.Enabled
		e.multiProcessExecs = make([]policy.ProcessConfig, len(p.Run.MultiProcess.Executables))
		copy(e.multiProcessExecs, p.Run.MultiProcess.Executables)
		e.testScenarios = make([]policy.MultiProcessScenario, len(p.Run.MultiProcess.TestScenarios))
		copy(e.testScenarios, p.Run.MultiProcess.TestScenarios)
	} else {
		e.multiProcessEnabled = false
		e.multiProcessExecs = []policy.ProcessConfig{}
		e.testScenarios = []policy.MultiProcessScenario{}
	}
	e.multiProcessCursor = 0
	e.testScenariosCursor = 0
}

func (e *Editor) SetWidth(w int) {
	e.width = w
}

func (e *Editor) Reset() {
	e.isNew = true
	e.filePath = ""
	e.focusedField = FieldName
	e.ErrorMsg = ""
	e.browsingForLibs = false
	e.showingExistingLibs = false
	e.existingLibs = nil
	e.existingLibsCursor = 0

	e.nameInput.SetValue("")
	e.nameInput.Focus()
	e.flagsInput.SetValue("-Wall -Wextra")
	e.sourceFileInput.SetValue("")
	e.libraryFiles = []string{}
	e.libraryFilesCursor = 0
	e.testFiles = []string{}
	e.testFilesCursor = 0
	e.browsingForTests = false
	e.showingExistingTests = false
	e.browsingForExpectedOutput = false
	e.showingExistingExpectedOutput = false
	e.existingExpectedOutputs = nil
	e.existingExpectedOutputsCursor = 0
	e.browsingForScenarioExpectedOutput = false
	e.showingExistingScenarioExpectedOutput = false
	e.scenarioExpectedOutputProcess = ""

	e.testCases = []policy.TestCase{}
	e.testCasesCursor = 0
	e.editingTestCase = false
	e.editingTestCaseIdx = -1
	e.resetTestCaseInputs()

	e.multiProcessEnabled = false
	e.multiProcessExecs = []policy.ProcessConfig{}
	e.multiProcessCursor = 0
	e.editingProcess = false
	e.editingProcessIdx = -1
	e.resetProcessInputs()

	e.testScenarios = []policy.MultiProcessScenario{}
	e.testScenariosCursor = 0
	e.editingScenario = false
	e.editingScenarioIdx = -1
	e.resetScenarioInputs()
}

func (e *Editor) resetTestCaseInputs() {
	e.testCaseInputs.name.SetValue("")
	e.testCaseInputs.args.SetValue("")
	e.testCaseInputs.input.SetValue("")
	e.testCaseInputs.expectedExit.SetValue("0")
	e.testCaseInputs.expectedOutputFile = ""
	e.testCaseInputs.focusedInput = 0
	e.testCaseInputs.name.Focus()
	e.testCaseInputs.args.Blur()
	e.testCaseInputs.input.Blur()
	e.testCaseInputs.expectedExit.Blur()
}

func (e *Editor) resetProcessInputs() {
	e.processInputs.name.SetValue("")
	e.processInputs.sourceFile.SetValue("")
	e.processInputs.args.SetValue("")
	e.processInputs.delayMs.SetValue("0")
	e.processInputs.focusedIdx = 0
	e.processInputs.name.Focus()
	e.processInputs.sourceFile.Blur()
	e.processInputs.args.Blur()
	e.processInputs.delayMs.Blur()
	e.ErrorMsg = ""
}

func (e *Editor) resetScenarioInputs() {
	e.scenarioInputs.name.SetValue("")
	e.scenarioInputs.name.Focus()
	e.scenarioInputs.focusedIdx = 0
	e.scenarioInputs.processArgs = make(map[string]textinput.Model)
	e.scenarioInputs.processStdin = make(map[string]textinput.Model)
	e.scenarioInputs.processExit = make(map[string]textinput.Model)
	e.scenarioInputs.expectedOutputs = make(map[string]string)
	e.scenarioExpectedOutputProcess = ""
}

func (e *Editor) initScenarioProcessInputs() {
	e.scenarioInputs.processArgs = make(map[string]textinput.Model)
	e.scenarioInputs.processStdin = make(map[string]textinput.Model)
	e.scenarioInputs.processExit = make(map[string]textinput.Model)
	if e.scenarioInputs.expectedOutputs == nil {
		e.scenarioInputs.expectedOutputs = make(map[string]string)
	}

	for _, proc := range e.multiProcessExecs {
		argsInput := textinput.New()
		argsInput.Placeholder = "arg1 arg2"
		argsInput.CharLimit = 200
		argsInput.Width = 30
		e.scenarioInputs.processArgs[proc.Name] = argsInput

		stdinInput := textinput.New()
		stdinInput.Placeholder = "stdin (use \\n for newlines)"
		stdinInput.CharLimit = 500
		stdinInput.Width = 30
		e.scenarioInputs.processStdin[proc.Name] = stdinInput

		exitInput := textinput.New()
		exitInput.Placeholder = "0"
		exitInput.CharLimit = 5
		exitInput.Width = 10
		e.scenarioInputs.processExit[proc.Name] = exitInput
	}
}

func (e *Editor) blurAllScenarioInputs() {
	e.scenarioInputs.name.Blur()
	for name := range e.scenarioInputs.processArgs {
		input := e.scenarioInputs.processArgs[name]
		input.Blur()
		e.scenarioInputs.processArgs[name] = input
	}
	for name := range e.scenarioInputs.processStdin {
		input := e.scenarioInputs.processStdin[name]
		input.Blur()
		e.scenarioInputs.processStdin[name] = input
	}
	for name := range e.scenarioInputs.processExit {
		input := e.scenarioInputs.processExit[name]
		input.Blur()
		e.scenarioInputs.processExit[name] = input
	}
}

func (e *Editor) focusCurrentScenarioInput() {
	numProcesses := len(e.multiProcessExecs)
	totalFields := 1 + (numProcesses * 4) + 1 // name + (4 fields per process) + save

	if e.scenarioInputs.focusedIdx == 0 {
		e.scenarioInputs.name.Focus()
	} else if e.scenarioInputs.focusedIdx < totalFields-1 {
		fieldOffset := e.scenarioInputs.focusedIdx - 1
		procIdx := fieldOffset / 4
		fieldType := fieldOffset % 4

		if procIdx < len(e.multiProcessExecs) {
			procName := e.multiProcessExecs[procIdx].Name
			switch fieldType {
			case 0: // args
				if input, ok := e.scenarioInputs.processArgs[procName]; ok {
					input.Focus()
					e.scenarioInputs.processArgs[procName] = input
				}
			case 1: // stdin
				if input, ok := e.scenarioInputs.processStdin[procName]; ok {
					input.Focus()
					e.scenarioInputs.processStdin[procName] = input
				}
			case 2: // exit
				if input, ok := e.scenarioInputs.processExit[procName]; ok {
					input.Focus()
					e.scenarioInputs.processExit[procName] = input
				}
			case 3: // expected output - display only, no text input to focus
			}
		}
	}
}

func (e *Editor) loadExistingTestFiles() {
	e.existingTests = nil
	testDir, err := config.TestFilesDir()
	if err != nil {
		return
	}

	entries, err := os.ReadDir(testDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		alreadyInPolicy := false
		for _, f := range e.testFiles {
			if f == name {
				alreadyInPolicy = true
				break
			}
		}
		if !alreadyInPolicy {
			e.existingTests = append(e.existingTests, name)
		}
	}
}

func (e *Editor) loadExistingExpectedOutputs() {
	e.existingExpectedOutputs = nil
	expDir, err := config.ExpectedOutputsDir()
	if err != nil {
		return
	}

	entries, err := os.ReadDir(expDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		e.existingExpectedOutputs = append(e.existingExpectedOutputs, entry.Name())
	}
}

func (e *Editor) copyToExpectedOutputs(selectedPath string) (string, bool) {
	filename := filepath.Base(selectedPath)
	expDir, err := config.EnsureExpectedOutputsDir()
	if err != nil {
		return "", false
	}
	destPath := filepath.Join(expDir, filename)
	data, err := os.ReadFile(selectedPath)
	if err != nil {
		return "", false
	}
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return "", false
	}
	return filename, true
}

func (e *Editor) handleExistingPicker(msg tea.KeyMsg, items []string, cursor *int, set func(string), close func()) {
	switch msg.String() {
	case "esc":
		close()
	case "j", "down":
		if *cursor < len(items)-1 {
			*cursor++
		}
	case "k", "up":
		if *cursor > 0 {
			*cursor--
		}
	case "enter":
		if *cursor < len(items) {
			set(items[*cursor])
		}
		close()
	}
}

func (e *Editor) renderBrowsePicker(title, subtitle string, useBox bool) string {
	var b strings.Builder
	b.WriteString(components.RenderHeader(title))

	if useBox {
		box := components.BoxStyle(60)
		var content strings.Builder
		if subtitle != "" {
			content.WriteString(components.SubtleText.Render(subtitle))
			content.WriteString("\n\n")
		}
		content.WriteString(e.folderBrowser.View())
		b.WriteString(box.Render(content.String()))
	} else {
		if subtitle != "" {
			b.WriteString(components.SubtleText.Render(subtitle))
			b.WriteString("\n\n")
		}
		b.WriteString(e.folderBrowser.View())
	}

	b.WriteString("\n\n")
	b.WriteString(components.SubtleText.Render("  enter select  •  esc cancel"))
	return b.String()
}

func (e *Editor) renderInputRow(label string, focused bool, input textinput.Model, hint string) string {
	var b strings.Builder
	b.WriteString(components.FocusPrefix(focused))
	b.WriteString(label)
	b.WriteString(input.View())
	if hint != "" {
		b.WriteString("\n")
		b.WriteString(components.SubtleText.Render(hint))
	}
	b.WriteString("\n\n")
	return b.String()
}

func (e *Editor) renderValueRow(label string, focused bool, value, empty string) string {
	var b strings.Builder
	b.WriteString(components.FocusPrefix(focused))
	b.WriteString(label)
	if value != "" {
		b.WriteString(components.SuccessText.Render(value))
	} else {
		b.WriteString(components.SubtleText.Render(empty))
	}
	b.WriteString("\n\n")
	return b.String()
}

func (e *Editor) renderInputRowTight(label string, focused bool, input textinput.Model) string {
	var b strings.Builder
	b.WriteString(components.FocusPrefix(focused))
	b.WriteString(label)
	b.WriteString(input.View())
	b.WriteString("\n")
	return b.String()
}

func (e *Editor) renderValueRowTight(label string, focused bool, value, empty string) string {
	var b strings.Builder
	b.WriteString(components.FocusPrefix(focused))
	b.WriteString(label)
	if value != "" {
		b.WriteString(components.SuccessText.Render(value))
	} else {
		b.WriteString(components.SubtleText.Render(empty))
	}
	b.WriteString("\n")
	return b.String()
}

func (e *Editor) renderExistingPicker(title, subtitle, emptyMsg string, items []string, cursor, boxWidth, maxVisible int, showCount bool) string {
	var b strings.Builder
	b.WriteString(components.RenderHeader(title))

	box := components.BoxStyle(boxWidth)
	var content strings.Builder
	if subtitle != "" {
		content.WriteString(components.SubtleText.Render(subtitle))
		content.WriteString("\n\n")
	}

	if len(items) == 0 {
		content.WriteString(components.SubtleText.Render(emptyMsg))
		if !strings.HasSuffix(emptyMsg, "\n") {
			content.WriteString("\n")
		}
	} else {
		start, end := e.getScrollWindow(cursor, len(items), maxVisible)
		for i := start; i < end; i++ {
			item := items[i]
			if i == cursor {
				content.WriteString("> " + components.SelectedItem.Render(item) + "\n")
			} else {
				content.WriteString("  " + components.NormalItem.Render(item) + "\n")
			}
		}
		if showCount && len(items) > maxVisible {
			content.WriteString(components.SubtleText.Render(fmt.Sprintf("\n  [%d-%d of %d]\n", start+1, end, len(items))))
		}
	}

	b.WriteString(box.Render(content.String()))
	b.WriteString("\n\n")
	b.WriteString(components.SubtleText.Render("  ↑↓ navigate  •  enter select  •  esc cancel"))
	return b.String()
}

func (e *Editor) loadExistingLibraries() {
	e.existingLibs = nil
	libDir, err := config.LibrariesDir()
	if err != nil {
		return
	}

	entries, err := os.ReadDir(libDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".c") || strings.HasSuffix(name, ".h") || strings.HasSuffix(name, ".o") {
			alreadyInPolicy := false
			for _, f := range e.libraryFiles {
				if f == name {
					alreadyInPolicy = true
					break
				}
			}
			if !alreadyInPolicy {
				e.existingLibs = append(e.existingLibs, name)
			}
		}
	}
}

func (e *Editor) Update(msg tea.Msg) tea.Cmd {
	// ─── SUB-MODE: Existing libraries picker ───
	if e.showingExistingLibs {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			e.handleExistingPicker(
				msg,
				e.existingLibs,
				&e.existingLibsCursor,
				func(item string) { e.libraryFiles = append(e.libraryFiles, item) },
				func() { e.showingExistingLibs = false },
			)
		}
		return nil
	}

	// ─── SUB-MODE: File browser for libraries ───
	if e.browsingForLibs {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				e.browsingForLibs = false
				return nil
			}

			selected, cmd := e.folderBrowser.Update(msg)
			if selected {
				selectedPath := e.folderBrowser.Selected()

				if strings.HasSuffix(selectedPath, ".c") || strings.HasSuffix(selectedPath, ".h") || strings.HasSuffix(selectedPath, ".o") {
					filename := filepath.Base(selectedPath)

					alreadyExists := false
					for _, f := range e.libraryFiles {
						if f == filename {
							alreadyExists = true
							break
						}
					}

					if !alreadyExists {
						libDir, err := config.EnsureLibrariesDir()
						if err == nil {
							destPath := filepath.Join(libDir, filename)
							data, err := os.ReadFile(selectedPath)
							if err == nil {
								if err := os.WriteFile(destPath, data, 0644); err == nil {
									e.libraryFiles = append(e.libraryFiles, filename)
								}
							}
						}
					}
				}

				e.browsingForLibs = false
				return nil
			}
			return cmd
		}
		return nil
	}

	// ─── SUB-MODE: File browser for test files ───
	if e.browsingForTests {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				e.browsingForTests = false
				return nil
			}

			selected, cmd := e.folderBrowser.Update(msg)
			if selected {
				selectedPath := e.folderBrowser.Selected()
				filename := filepath.Base(selectedPath)

				alreadyExists := false
				for _, f := range e.testFiles {
					if f == filename {
						alreadyExists = true
						break
					}
				}

				if !alreadyExists {
					testDir, err := config.EnsureTestFilesDir()
					if err == nil {
						destPath := filepath.Join(testDir, filename)
						data, err := os.ReadFile(selectedPath)
						if err == nil {
							if err := os.WriteFile(destPath, data, 0644); err == nil {
								e.testFiles = append(e.testFiles, filename)
							}
						}
					}
				}

				e.browsingForTests = false
				return nil
			}
			return cmd
		}
		return nil
	}

	// ─── SUB-MODE: Existing test files picker ───
	if e.showingExistingTests {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			e.handleExistingPicker(
				msg,
				e.existingTests,
				&e.existingTestsCursor,
				func(item string) { e.testFiles = append(e.testFiles, item) },
				func() { e.showingExistingTests = false },
			)
		}
		return nil
	}

	// ─── SUB-MODE: Browsing for expected output file ───
	if e.browsingForExpectedOutput {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "esc" {
				e.browsingForExpectedOutput = false
				return nil
			}

			selected, cmd := e.folderBrowser.Update(msg)
			if selected {
				if filename, ok := e.copyToExpectedOutputs(e.folderBrowser.Selected()); ok {
					e.testCaseInputs.expectedOutputFile = filename
				}
				e.browsingForExpectedOutput = false
				return nil
			}
			return cmd
		}
		return nil
	}

	// ─── SUB-MODE: Picking existing expected output file ───
	if e.showingExistingExpectedOutput {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			e.handleExistingPicker(
				msg,
				e.existingExpectedOutputs,
				&e.existingExpectedOutputsCursor,
				func(filename string) { e.testCaseInputs.expectedOutputFile = filename },
				func() { e.showingExistingExpectedOutput = false },
			)
		}
		return nil
	}

	// ─── SUB-MODE: Test case editor form ───
	if e.editingTestCase {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				e.editingTestCase = false
				e.editingTestCaseIdx = -1
				e.resetTestCaseInputs()
				return nil
			case "tab", "down":
				e.testCaseInputs.name.Blur()
				e.testCaseInputs.args.Blur()
				e.testCaseInputs.input.Blur()
				e.testCaseInputs.expectedExit.Blur()
				e.testCaseInputs.focusedInput = (e.testCaseInputs.focusedInput + 1) % 6
				switch e.testCaseInputs.focusedInput {
				case 0:
					e.testCaseInputs.name.Focus()
				case 1:
					e.testCaseInputs.args.Focus()
				case 2:
					e.testCaseInputs.input.Focus()
				case 3:
					e.testCaseInputs.expectedExit.Focus()
				}
				return nil
			case "shift+tab", "up":
				e.testCaseInputs.name.Blur()
				e.testCaseInputs.args.Blur()
				e.testCaseInputs.input.Blur()
				e.testCaseInputs.expectedExit.Blur()
				e.testCaseInputs.focusedInput = (e.testCaseInputs.focusedInput + 5) % 6
				switch e.testCaseInputs.focusedInput {
				case 0:
					e.testCaseInputs.name.Focus()
				case 1:
					e.testCaseInputs.args.Focus()
				case 2:
					e.testCaseInputs.input.Focus()
				case 3:
					e.testCaseInputs.expectedExit.Focus()
				}
				return nil
			case "a":
				if e.testCaseInputs.focusedInput == 4 {
					cwd, _ := os.Getwd()
					e.folderBrowser.Reset(cwd)
					e.folderBrowser.SetFileMode(true)
					e.folderBrowser.SetFileExtensions([]string{".txt", ".out", ".expected", ".log"})
					e.browsingForExpectedOutput = true
					return nil
				}
			case "e":
				if e.testCaseInputs.focusedInput == 4 {
					e.loadExistingExpectedOutputs()
					e.existingExpectedOutputsCursor = 0
					if len(e.existingExpectedOutputs) > 0 {
						e.showingExistingExpectedOutput = true
					}
					return nil
				}
			case "d":
				if e.testCaseInputs.focusedInput == 4 {
					e.testCaseInputs.expectedOutputFile = ""
					return nil
				}
			case "enter":
				if e.testCaseInputs.focusedInput == 5 {
					tc := policy.TestCase{
						Name:               e.testCaseInputs.name.Value(),
						Input:              e.testCaseInputs.input.Value(),
						ExpectedOutputFile: e.testCaseInputs.expectedOutputFile,
					}
					if tc.Name == "" {
						tc.Name = fmt.Sprintf("Test %d", len(e.testCases)+1)
					}
					if args := e.testCaseInputs.args.Value(); args != "" {
						tc.Args = strings.Fields(args)
					}
					if exitStr := e.testCaseInputs.expectedExit.Value(); exitStr != "" {
						var exitCode int
						if _, err := fmt.Sscanf(exitStr, "%d", &exitCode); err == nil {
							tc.ExpectedExit = &exitCode
						}
					}

					if e.editingTestCaseIdx >= 0 && e.editingTestCaseIdx < len(e.testCases) {
						e.testCases[e.editingTestCaseIdx] = tc
					} else {
						e.testCases = append(e.testCases, tc)
					}
					e.editingTestCase = false
					e.editingTestCaseIdx = -1
					e.resetTestCaseInputs()
					return nil
				}
			}

			var cmd tea.Cmd
			switch e.testCaseInputs.focusedInput {
			case 0:
				e.testCaseInputs.name, cmd = e.testCaseInputs.name.Update(msg)
			case 1:
				e.testCaseInputs.args, cmd = e.testCaseInputs.args.Update(msg)
			case 2:
				e.testCaseInputs.input, cmd = e.testCaseInputs.input.Update(msg)
			case 3:
				e.testCaseInputs.expectedExit, cmd = e.testCaseInputs.expectedExit.Update(msg)
			}
			return cmd
		}
		return nil
	}

	// ─── SUB-MODE: Process config editor form ───
	if e.editingProcess {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				e.editingProcess = false
				e.editingProcessIdx = -1
				e.resetProcessInputs()
				return nil
			case "tab", "down":
				e.processInputs.name.Blur()
				e.processInputs.sourceFile.Blur()
				e.processInputs.args.Blur()
				e.processInputs.delayMs.Blur()
				e.processInputs.focusedIdx = (e.processInputs.focusedIdx + 1) % 5
				switch e.processInputs.focusedIdx {
				case 0:
					e.processInputs.name.Focus()
				case 1:
					e.processInputs.sourceFile.Focus()
				case 2:
					e.processInputs.args.Focus()
				case 3:
					e.processInputs.delayMs.Focus()
				}
				return nil
			case "shift+tab", "up":
				e.processInputs.name.Blur()
				e.processInputs.sourceFile.Blur()
				e.processInputs.args.Blur()
				e.processInputs.delayMs.Blur()
				e.processInputs.focusedIdx = (e.processInputs.focusedIdx + 4) % 5
				switch e.processInputs.focusedIdx {
				case 0:
					e.processInputs.name.Focus()
				case 1:
					e.processInputs.sourceFile.Focus()
				case 2:
					e.processInputs.args.Focus()
				case 3:
					e.processInputs.delayMs.Focus()
				}
				return nil
			case "enter":
				if e.processInputs.focusedIdx == 4 {
					proc := policy.ProcessConfig{
						Name:       e.processInputs.name.Value(),
						SourceFile: e.processInputs.sourceFile.Value(),
					}
					if proc.Name == "" {
						proc.Name = fmt.Sprintf("Process %d", len(e.multiProcessExecs)+1)
					}
					if args := e.processInputs.args.Value(); args != "" {
						proc.Args = strings.Fields(args)
					}
					if delayStr := e.processInputs.delayMs.Value(); delayStr != "" {
						var delay int
						if _, err := fmt.Sscanf(delayStr, "%d", &delay); err == nil {
							proc.StartDelayMs = delay
						}
					}

					if e.editingProcessIdx >= 0 && e.editingProcessIdx < len(e.multiProcessExecs) {
						e.multiProcessExecs[e.editingProcessIdx] = proc
					} else {
						e.multiProcessExecs = append(e.multiProcessExecs, proc)
						e.multiProcessEnabled = true
					}
					e.editingProcess = false
					e.editingProcessIdx = -1
					e.resetProcessInputs()
					return nil
				}
			}

			var cmd tea.Cmd
			switch e.processInputs.focusedIdx {
			case 0:
				e.processInputs.name, cmd = e.processInputs.name.Update(msg)
			case 1:
				e.processInputs.sourceFile, cmd = e.processInputs.sourceFile.Update(msg)
			case 2:
				e.processInputs.args, cmd = e.processInputs.args.Update(msg)
			case 3:
				e.processInputs.delayMs, cmd = e.processInputs.delayMs.Update(msg)
			}
			return cmd
		}
		return nil
	}

	// ─── SUB-MODE: Browsing for scenario expected output file ───
	if e.browsingForScenarioExpectedOutput {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "esc" {
				e.browsingForScenarioExpectedOutput = false
				e.scenarioExpectedOutputProcess = ""
				return nil
			}

			selected, cmd := e.folderBrowser.Update(msg)
			if selected {
				if filename, ok := e.copyToExpectedOutputs(e.folderBrowser.Selected()); ok {
					e.scenarioInputs.expectedOutputs[e.scenarioExpectedOutputProcess] = filename
				}
				e.browsingForScenarioExpectedOutput = false
				e.scenarioExpectedOutputProcess = ""
				return nil
			}
			return cmd
		}
		return nil
	}

	// ─── SUB-MODE: Selecting existing expected output for scenario ───
	if e.showingExistingScenarioExpectedOutput {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			e.handleExistingPicker(
				msg,
				e.existingExpectedOutputs,
				&e.existingExpectedOutputsCursor,
				func(filename string) { e.scenarioInputs.expectedOutputs[e.scenarioExpectedOutputProcess] = filename },
				func() {
					e.showingExistingScenarioExpectedOutput = false
					e.scenarioExpectedOutputProcess = ""
				},
			)
		}
		return nil
	}

	// ─── SUB-MODE: Test scenario editor form ───
	if e.editingScenario {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			numProcesses := len(e.multiProcessExecs)
			totalFields := 1 + (numProcesses * 4) + 1 // name + (4 fields per process) + save

			// Check if we're on an expected output field
			isOnExpectedOutput := false
			var currentProcName string
			if e.scenarioInputs.focusedIdx > 0 && e.scenarioInputs.focusedIdx < totalFields-1 {
				fieldOffset := e.scenarioInputs.focusedIdx - 1
				procIdx := fieldOffset / 4
				fieldType := fieldOffset % 4
				if fieldType == 3 && procIdx < len(e.multiProcessExecs) {
					isOnExpectedOutput = true
					currentProcName = e.multiProcessExecs[procIdx].Name
				}
			}

			switch msg.String() {
			case "esc":
				e.editingScenario = false
				e.editingScenarioIdx = -1
				e.resetScenarioInputs()
				return nil
			case "tab", "down":
				e.blurAllScenarioInputs()
				e.scenarioInputs.focusedIdx = (e.scenarioInputs.focusedIdx + 1) % totalFields
				e.focusCurrentScenarioInput()
				return nil
			case "shift+tab", "up":
				e.blurAllScenarioInputs()
				e.scenarioInputs.focusedIdx = (e.scenarioInputs.focusedIdx + totalFields - 1) % totalFields
				e.focusCurrentScenarioInput()
				return nil
			case "a":
				if isOnExpectedOutput {
					cwd, _ := os.Getwd()
					e.folderBrowser.Reset(cwd)
					e.folderBrowser.SetFileMode(true)
					e.folderBrowser.SetFileExtensions([]string{".txt", ".out", ".expected", ".log"})
					e.browsingForScenarioExpectedOutput = true
					e.scenarioExpectedOutputProcess = currentProcName
					return nil
				}
			case "e":
				if isOnExpectedOutput {
					e.loadExistingExpectedOutputs()
					if len(e.existingExpectedOutputs) > 0 {
						e.showingExistingScenarioExpectedOutput = true
						e.scenarioExpectedOutputProcess = currentProcName
						e.existingExpectedOutputsCursor = 0
					}
					return nil
				}
			case "d":
				if isOnExpectedOutput {
					delete(e.scenarioInputs.expectedOutputs, currentProcName)
					return nil
				}
			case "enter":
				saveIdx := totalFields - 1
				if e.scenarioInputs.focusedIdx == saveIdx {
					scenario := policy.MultiProcessScenario{
						Name:            e.scenarioInputs.name.Value(),
						ProcessArgs:     make(map[string][]string),
						ProcessInputs:   make(map[string]string),
						ExpectedExits:   make(map[string]int),
						ExpectedOutputs: make(map[string]string),
					}
					if scenario.Name == "" {
						scenario.Name = fmt.Sprintf("Scenario %d", len(e.testScenarios)+1)
					}

					for _, proc := range e.multiProcessExecs {
						if argsInput, ok := e.scenarioInputs.processArgs[proc.Name]; ok {
							if args := argsInput.Value(); args != "" {
								scenario.ProcessArgs[proc.Name] = strings.Fields(args)
							}
						}
						if stdinInput, ok := e.scenarioInputs.processStdin[proc.Name]; ok {
							if stdin := stdinInput.Value(); stdin != "" {
								scenario.ProcessInputs[proc.Name] = stdin
							}
						}
						if exitInput, ok := e.scenarioInputs.processExit[proc.Name]; ok {
							if exitStr := exitInput.Value(); exitStr != "" {
								var exitCode int
								if _, err := fmt.Sscanf(exitStr, "%d", &exitCode); err == nil {
									scenario.ExpectedExits[proc.Name] = exitCode
								}
							}
						}
						if expOut, ok := e.scenarioInputs.expectedOutputs[proc.Name]; ok && expOut != "" {
							scenario.ExpectedOutputs[proc.Name] = expOut
						}
					}

					if e.editingScenarioIdx >= 0 && e.editingScenarioIdx < len(e.testScenarios) {
						e.testScenarios[e.editingScenarioIdx] = scenario
					} else {
						e.testScenarios = append(e.testScenarios, scenario)
					}
					e.editingScenario = false
					e.editingScenarioIdx = -1
					e.resetScenarioInputs()
					return nil
				}
			}

			// Handle text input updates (only for text input fields, not expected output)
			var cmd tea.Cmd
			if e.scenarioInputs.focusedIdx == 0 {
				e.scenarioInputs.name, cmd = e.scenarioInputs.name.Update(msg)
			} else if e.scenarioInputs.focusedIdx < totalFields-1 && !isOnExpectedOutput {
				fieldOffset := e.scenarioInputs.focusedIdx - 1
				procIdx := fieldOffset / 4
				fieldType := fieldOffset % 4

				if procIdx < len(e.multiProcessExecs) {
					procName := e.multiProcessExecs[procIdx].Name
					switch fieldType {
					case 0: // args
						if input, ok := e.scenarioInputs.processArgs[procName]; ok {
							e.scenarioInputs.processArgs[procName], cmd = input.Update(msg)
						}
					case 1: // stdin
						if input, ok := e.scenarioInputs.processStdin[procName]; ok {
							e.scenarioInputs.processStdin[procName], cmd = input.Update(msg)
						}
					case 2: // exit
						if input, ok := e.scenarioInputs.processExit[procName]; ok {
							e.scenarioInputs.processExit[procName], cmd = input.Update(msg)
						}
					}
				}
			}
			return cmd
		}
		return nil
	}

	// ─── MAIN FORM: Field-specific key handling ───
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if e.focusedField == FieldMultiProcess {
			switch msg.String() {
			case "a":
				e.editingProcess = true
				e.editingProcessIdx = -1
				e.resetProcessInputs()
				return nil
			case "enter":
				if len(e.multiProcessExecs) > 0 && e.multiProcessCursor < len(e.multiProcessExecs) {
					proc := e.multiProcessExecs[e.multiProcessCursor]
					e.editingProcess = true
					e.editingProcessIdx = e.multiProcessCursor
					e.processInputs.name.SetValue(proc.Name)
					e.processInputs.sourceFile.SetValue(proc.SourceFile)
					e.processInputs.args.SetValue(strings.Join(proc.Args, " "))
					e.processInputs.delayMs.SetValue(fmt.Sprintf("%d", proc.StartDelayMs))
					e.processInputs.focusedIdx = 0
					e.processInputs.name.Focus()
				}
				return nil
			case "d", "backspace":
				if len(e.multiProcessExecs) > 0 && e.multiProcessCursor < len(e.multiProcessExecs) {
					e.multiProcessExecs = append(e.multiProcessExecs[:e.multiProcessCursor], e.multiProcessExecs[e.multiProcessCursor+1:]...)
					if e.multiProcessCursor >= len(e.multiProcessExecs) && e.multiProcessCursor > 0 {
						e.multiProcessCursor--
					}
					if len(e.multiProcessExecs) == 0 {
						e.multiProcessEnabled = false
					}
				}
				return nil
			case "e":
				e.multiProcessEnabled = !e.multiProcessEnabled
				return nil
			case "j", "down":
				if len(e.multiProcessExecs) > 0 && e.multiProcessCursor < len(e.multiProcessExecs)-1 {
					e.multiProcessCursor++
				} else {
					e.nextField()
				}
				return nil
			case "k", "up":
				if len(e.multiProcessExecs) > 0 && e.multiProcessCursor > 0 {
					e.multiProcessCursor--
				} else {
					e.prevField()
				}
				return nil
			case "tab":
				e.nextField()
				return nil
			case "shift+tab":
				e.prevField()
				return nil
			}
			return nil
		}

		if e.focusedField == FieldMultiProcessTests {
			switch msg.String() {
			case "a":
				if len(e.multiProcessExecs) > 0 {
					e.editingScenario = true
					e.editingScenarioIdx = -1
					e.resetScenarioInputs()
					e.initScenarioProcessInputs()
					e.scenarioInputs.name.Focus()
				}
				return nil
			case "enter":
				if len(e.testScenarios) > 0 && e.testScenariosCursor < len(e.testScenarios) {
					scenario := e.testScenarios[e.testScenariosCursor]
					e.editingScenario = true
					e.editingScenarioIdx = e.testScenariosCursor
					e.initScenarioProcessInputs()
					e.scenarioInputs.name.SetValue(scenario.Name)
					for _, proc := range e.multiProcessExecs {
						if args, ok := scenario.ProcessArgs[proc.Name]; ok {
							if input, exists := e.scenarioInputs.processArgs[proc.Name]; exists {
								input.SetValue(strings.Join(args, " "))
								e.scenarioInputs.processArgs[proc.Name] = input
							}
						}
						if stdin, ok := scenario.ProcessInputs[proc.Name]; ok {
							if input, exists := e.scenarioInputs.processStdin[proc.Name]; exists {
								input.SetValue(stdin)
								e.scenarioInputs.processStdin[proc.Name] = input
							}
						}
						if exit, ok := scenario.ExpectedExits[proc.Name]; ok {
							if input, exists := e.scenarioInputs.processExit[proc.Name]; exists {
								input.SetValue(fmt.Sprintf("%d", exit))
								e.scenarioInputs.processExit[proc.Name] = input
							}
						}
						if expOut, ok := scenario.ExpectedOutputs[proc.Name]; ok {
							e.scenarioInputs.expectedOutputs[proc.Name] = expOut
						}
					}
					e.scenarioInputs.focusedIdx = 0
					e.scenarioInputs.name.Focus()
				}
				return nil
			case "d", "backspace":
				if len(e.testScenarios) > 0 && e.testScenariosCursor < len(e.testScenarios) {
					e.testScenarios = append(e.testScenarios[:e.testScenariosCursor], e.testScenarios[e.testScenariosCursor+1:]...)
					if e.testScenariosCursor >= len(e.testScenarios) && e.testScenariosCursor > 0 {
						e.testScenariosCursor--
					}
				}
				return nil
			case "j", "down":
				if len(e.testScenarios) > 0 && e.testScenariosCursor < len(e.testScenarios)-1 {
					e.testScenariosCursor++
				} else {
					e.nextField()
				}
				return nil
			case "k", "up":
				if len(e.testScenarios) > 0 && e.testScenariosCursor > 0 {
					e.testScenariosCursor--
				} else {
					e.prevField()
				}
				return nil
			case "tab":
				e.nextField()
				return nil
			case "shift+tab":
				e.prevField()
				return nil
			}
			return nil
		}

		if e.focusedField == FieldMultiProcessToggle {
			switch msg.String() {
			case "e", "enter", " ":
				e.multiProcessEnabled = !e.multiProcessEnabled
				return nil
			case "tab":
				e.nextField()
				return nil
			case "shift+tab":
				e.prevField()
				return nil
			case "j", "down":
				e.nextField()
				return nil
			case "k", "up":
				e.prevField()
				return nil
			}
			return nil
		}

		if e.focusedField == FieldTestCases {
			switch msg.String() {
			case "a":
				e.editingTestCase = true
				e.editingTestCaseIdx = -1
				e.resetTestCaseInputs()
				return nil
			case "enter":
				if len(e.testCases) > 0 && e.testCasesCursor < len(e.testCases) {
					tc := e.testCases[e.testCasesCursor]
					e.editingTestCase = true
					e.editingTestCaseIdx = e.testCasesCursor
					e.testCaseInputs.name.SetValue(tc.Name)
					e.testCaseInputs.args.SetValue(strings.Join(tc.Args, " "))
					e.testCaseInputs.input.SetValue(tc.Input)
					if tc.ExpectedExit != nil {
						e.testCaseInputs.expectedExit.SetValue(fmt.Sprintf("%d", *tc.ExpectedExit))
					} else {
						e.testCaseInputs.expectedExit.SetValue("0")
					}
					e.testCaseInputs.expectedOutputFile = tc.ExpectedOutputFile
					e.testCaseInputs.focusedInput = 0
					e.testCaseInputs.name.Focus()
				}
				return nil
			case "d", "backspace":
				if len(e.testCases) > 0 && e.testCasesCursor < len(e.testCases) {
					e.testCases = append(e.testCases[:e.testCasesCursor], e.testCases[e.testCasesCursor+1:]...)
					if e.testCasesCursor >= len(e.testCases) && e.testCasesCursor > 0 {
						e.testCasesCursor--
					}
				}
				return nil
			case "j", "down":
				if len(e.testCases) > 0 && e.testCasesCursor < len(e.testCases)-1 {
					e.testCasesCursor++
				} else {
					e.nextField()
				}
				return nil
			case "k", "up":
				if len(e.testCases) > 0 && e.testCasesCursor > 0 {
					e.testCasesCursor--
				} else {
					e.prevField()
				}
				return nil
			case "tab":
				e.nextField()
				return nil
			case "shift+tab":
				e.prevField()
				return nil
			}
			return nil
		}

		if e.focusedField == FieldLibraryFiles {
			switch msg.String() {
			case "a":
				cwd, _ := os.Getwd()
				e.folderBrowser.Reset(cwd)
				e.folderBrowser.SetFileMode(true)
				e.folderBrowser.SetFileExtensions([]string{".c", ".h", ".o"})
				e.browsingForLibs = true
				return nil
			case "e":
				e.loadExistingLibraries()
				e.existingLibsCursor = 0
				if len(e.existingLibs) > 0 {
					e.showingExistingLibs = true
				}
				return nil
			case "d", "backspace":
				if len(e.libraryFiles) > 0 && e.libraryFilesCursor < len(e.libraryFiles) {
					e.libraryFiles = append(e.libraryFiles[:e.libraryFilesCursor], e.libraryFiles[e.libraryFilesCursor+1:]...)
					if e.libraryFilesCursor >= len(e.libraryFiles) && e.libraryFilesCursor > 0 {
						e.libraryFilesCursor--
					}
				}
				return nil
			case "j", "down":
				if len(e.libraryFiles) > 0 && e.libraryFilesCursor < len(e.libraryFiles)-1 {
					e.libraryFilesCursor++
				} else {
					e.nextField()
				}
				return nil
			case "k", "up":
				if len(e.libraryFiles) > 0 && e.libraryFilesCursor > 0 {
					e.libraryFilesCursor--
				} else {
					e.prevField()
				}
				return nil
			case "tab":
				e.nextField()
				return nil
			case "shift+tab":
				e.prevField()
				return nil
			}
			return nil
		}

		if e.focusedField == FieldTestFiles {
			switch msg.String() {
			case "a":
				cwd, _ := os.Getwd()
				e.folderBrowser.Reset(cwd)
				e.folderBrowser.SetFileMode(true)
				e.folderBrowser.SetFileExtensions([]string{".txt", ".bin", ".dat", ".in", ".out"})
				e.browsingForTests = true
				return nil
			case "e":
				e.loadExistingTestFiles()
				e.existingTestsCursor = 0
				if len(e.existingTests) > 0 {
					e.showingExistingTests = true
				}
				return nil
			case "d", "backspace":
				if len(e.testFiles) > 0 && e.testFilesCursor < len(e.testFiles) {
					e.testFiles = append(e.testFiles[:e.testFilesCursor], e.testFiles[e.testFilesCursor+1:]...)
					if e.testFilesCursor >= len(e.testFiles) && e.testFilesCursor > 0 {
						e.testFilesCursor--
					}
				}
				return nil
			case "j", "down":
				if len(e.testFiles) > 0 && e.testFilesCursor < len(e.testFiles)-1 {
					e.testFilesCursor++
				} else {
					e.nextField()
				}
				return nil
			case "k", "up":
				if len(e.testFiles) > 0 && e.testFilesCursor > 0 {
					e.testFilesCursor--
				} else {
					e.prevField()
				}
				return nil
			case "tab":
				e.nextField()
				return nil
			case "shift+tab":
				e.prevField()
				return nil
			}
			return nil
		}

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

	var cmd tea.Cmd
	switch e.focusedField {
	case FieldName:
		e.nameInput, cmd = e.nameInput.Update(msg)
	case FieldFlags:
		e.flagsInput, cmd = e.flagsInput.Update(msg)
	case FieldSourceFile:
		e.sourceFileInput, cmd = e.sourceFileInput.Update(msg)
	}

	return cmd
}

func (e *Editor) nextField() {
	e.blurAll()
	e.focusedField++
	e.focusedField = e.adjustFieldForMode(e.focusedField, true)
	if e.focusedField > FieldCancel {
		e.focusedField = FieldName
	}
	e.focusCurrent()
}

func (e *Editor) prevField() {
	e.blurAll()
	if e.focusedField == FieldName {
		e.focusedField = FieldCancel
	} else {
		e.focusedField--
		e.focusedField = e.adjustFieldForMode(e.focusedField, false)
	}
	e.focusCurrent()
}

func (e *Editor) adjustFieldForMode(field EditorField, forward bool) EditorField {
	if e.multiProcessEnabled {
		if field == FieldSourceFile || field == FieldTestCases {
			if forward {
				return FieldMultiProcess
			}
			return FieldMultiProcessToggle
		}
	} else {
		if field == FieldMultiProcess || field == FieldMultiProcessTests {
			if forward {
				return FieldSave
			}
			return FieldTestCases
		}
	}
	return field
}

func (e *Editor) blurAll() {
	e.nameInput.Blur()
	e.flagsInput.Blur()
	e.sourceFileInput.Blur()
}

func (e *Editor) focusCurrent() {
	switch e.focusedField {
	case FieldName:
		e.nameInput.Focus()
	case FieldFlags:
		e.flagsInput.Focus()
	case FieldSourceFile:
		e.sourceFileInput.Focus()
	}
}

func (e *Editor) save() tea.Cmd {
	return func() tea.Msg {
		name := strings.TrimSpace(e.nameInput.Value())
		if name == "" {
			return SaveErrorMsg{Err: "Policy name is required"}
		}

		flagsStr := strings.TrimSpace(e.flagsInput.Value())
		var flags []string
		if flagsStr != "" {
			flags = strings.Fields(flagsStr)
		}

		sourceFile := strings.TrimSpace(e.sourceFileInput.Value())

		type TestCaseYAML struct {
			Name               string   `yaml:"name,omitempty"`
			Args               []string `yaml:"args,omitempty"`
			Input              string   `yaml:"input,omitempty"`
			ExpectedExit       *int     `yaml:"expected_exit,omitempty"`
			ExpectedOutputFile string   `yaml:"expected_output_file,omitempty"`
		}

		p := struct {
			Name    string `yaml:"name"`
			Compile struct {
				GCC        string   `yaml:"gcc"`
				Flags      []string `yaml:"flags"`
				SourceFile string   `yaml:"source_file,omitempty"`
			} `yaml:"compile"`
			Run struct {
				TestCases    []TestCaseYAML `yaml:"test_cases,omitempty"`
				MultiProcess *struct {
					Enabled     bool `yaml:"enabled"`
					Executables []struct {
						Name         string   `yaml:"name"`
						SourceFile   string   `yaml:"source_file"`
						Args         []string `yaml:"args,omitempty"`
						Input        string   `yaml:"input,omitempty"`
						StartDelayMs int      `yaml:"start_delay_ms,omitempty"`
					} `yaml:"executables"`
					TestScenarios []struct {
						Name            string              `yaml:"name"`
						ProcessArgs     map[string][]string `yaml:"process_args,omitempty"`
						ProcessInputs   map[string]string   `yaml:"process_inputs,omitempty"`
						ExpectedExits   map[string]int      `yaml:"expected_exits,omitempty"`
						ExpectedOutputs map[string]string   `yaml:"expected_outputs,omitempty"`
					} `yaml:"test_scenarios,omitempty"`
				} `yaml:"multi_process,omitempty"`
			} `yaml:"run,omitempty"`
			LibraryFiles []string `yaml:"library_files,omitempty"`
			TestFiles    []string `yaml:"test_files,omitempty"`
		}{}

		p.Name = name
		p.Compile.GCC = "gcc"
		p.Compile.Flags = flags
		p.Compile.SourceFile = sourceFile
		p.LibraryFiles = e.libraryFiles
		p.TestFiles = e.testFiles

		if len(e.testCases) > 0 || len(e.multiProcessExecs) > 0 {
			for _, tc := range e.testCases {
				p.Run.TestCases = append(p.Run.TestCases, TestCaseYAML{
					Name:               tc.Name,
					Args:               tc.Args,
					Input:              tc.Input,
					ExpectedExit:       tc.ExpectedExit,
					ExpectedOutputFile: tc.ExpectedOutputFile,
				})
			}

			if e.multiProcessEnabled && len(e.multiProcessExecs) > 0 {
				p.Run.MultiProcess = &struct {
					Enabled     bool `yaml:"enabled"`
					Executables []struct {
						Name         string   `yaml:"name"`
						SourceFile   string   `yaml:"source_file"`
						Args         []string `yaml:"args,omitempty"`
						Input        string   `yaml:"input,omitempty"`
						StartDelayMs int      `yaml:"start_delay_ms,omitempty"`
					} `yaml:"executables"`
					TestScenarios []struct {
						Name            string              `yaml:"name"`
						ProcessArgs     map[string][]string `yaml:"process_args,omitempty"`
						ProcessInputs   map[string]string   `yaml:"process_inputs,omitempty"`
						ExpectedExits   map[string]int      `yaml:"expected_exits,omitempty"`
						ExpectedOutputs map[string]string   `yaml:"expected_outputs,omitempty"`
					} `yaml:"test_scenarios,omitempty"`
				}{
					Enabled: true,
				}
				for _, proc := range e.multiProcessExecs {
					p.Run.MultiProcess.Executables = append(p.Run.MultiProcess.Executables, struct {
						Name         string   `yaml:"name"`
						SourceFile   string   `yaml:"source_file"`
						Args         []string `yaml:"args,omitempty"`
						Input        string   `yaml:"input,omitempty"`
						StartDelayMs int      `yaml:"start_delay_ms,omitempty"`
					}{
						Name:         proc.Name,
						SourceFile:   proc.SourceFile,
						Args:         proc.Args,
						Input:        proc.Input,
						StartDelayMs: proc.StartDelayMs,
					})
				}
				for _, scenario := range e.testScenarios {
					p.Run.MultiProcess.TestScenarios = append(p.Run.MultiProcess.TestScenarios, struct {
						Name            string              `yaml:"name"`
						ProcessArgs     map[string][]string `yaml:"process_args,omitempty"`
						ProcessInputs   map[string]string   `yaml:"process_inputs,omitempty"`
						ExpectedExits   map[string]int      `yaml:"expected_exits,omitempty"`
						ExpectedOutputs map[string]string   `yaml:"expected_outputs,omitempty"`
					}{
						Name:            scenario.Name,
						ProcessArgs:     scenario.ProcessArgs,
						ProcessInputs:   scenario.ProcessInputs,
						ExpectedExits:   scenario.ExpectedExits,
						ExpectedOutputs: scenario.ExpectedOutputs,
					})
				}
			}
		}

		data, err := yaml.Marshal(p)
		if err != nil {
			return SaveErrorMsg{Err: fmt.Sprintf("Failed to create YAML: %v", err)}
		}

		var filePath string
		if e.isNew {
			safeName := strings.ToLower(name)
			safeName = strings.ReplaceAll(safeName, " ", "-")
			safeName = strings.ReplaceAll(safeName, "/", "-")
			safeName = strings.ReplaceAll(safeName, "\\", "-")

			policiesDir, err := config.PoliciesDir()
			if err != nil {
				return SaveErrorMsg{Err: fmt.Sprintf("Failed to get config dir: %v", err)}
			}

			if err := os.MkdirAll(policiesDir, 0755); err != nil {
				return SaveErrorMsg{Err: fmt.Sprintf("Failed to create directory: %v", err)}
			}

			filePath = filepath.Join(policiesDir, safeName+".yaml")
		} else {
			filePath = e.filePath
		}

		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return SaveErrorMsg{Err: fmt.Sprintf("Failed to save: %v", err)}
		}

		return SavedMsg{Path: filePath, IsNew: e.isNew}
	}
}

func (e *Editor) View() string {
	if e.editingProcess {
		var b strings.Builder
		if e.editingProcessIdx >= 0 {
			b.WriteString(components.RenderHeader("Edit Process"))
		} else {
			b.WriteString(components.RenderHeader("Add Process"))
		}

		box := components.BoxStyle(80)
		var content strings.Builder

		content.WriteString(components.FocusPrefix(e.processInputs.focusedIdx == 0))
		content.WriteString("Name:      ")
		content.WriteString(e.processInputs.name.View())
		content.WriteString("\n\n")

		content.WriteString(components.FocusPrefix(e.processInputs.focusedIdx == 1))
		content.WriteString("Source:    ")
		content.WriteString(e.processInputs.sourceFile.View())
		content.WriteString("\n")
		content.WriteString(components.SubtleText.Render("             (binary = filename without .c)"))
		content.WriteString("\n\n")

		content.WriteString(components.FocusPrefix(e.processInputs.focusedIdx == 2))
		content.WriteString("Arguments: ")
		content.WriteString(e.processInputs.args.View())
		content.WriteString("\n\n")

		content.WriteString(components.FocusPrefix(e.processInputs.focusedIdx == 3))
		content.WriteString("Delay (ms):")
		content.WriteString(e.processInputs.delayMs.View())
		content.WriteString("\n\n")

		buttonText := "[ Add Process ]"
		if e.editingProcessIdx >= 0 {
			buttonText = "[ Save Changes ]"
		}
		content.WriteString(components.FocusPrefix(e.processInputs.focusedIdx == 4))
		if e.processInputs.focusedIdx == 4 {
			content.WriteString(components.SelectedItem.Render(buttonText))
		} else {
			content.WriteString(components.NormalItem.Render(buttonText))
		}

		b.WriteString(box.Render(content.String()))
		b.WriteString("\n\n")
		if e.ErrorMsg != "" {
			b.WriteString(components.ErrorStyle.Render("  Error: " + e.ErrorMsg))
			b.WriteString("\n")
		}
		b.WriteString(components.SubtleText.Render("  tab/↑↓ navigate  •  enter save  •  esc cancel"))

		return b.String()
	}

	if e.browsingForExpectedOutput {
		return e.renderBrowsePicker("Select Expected Output File", "", false)
	}

	if e.showingExistingExpectedOutput {
		return e.renderExistingPicker(
			"Select Existing Expected Output",
			"",
			"No expected output files found.\nAdd files to ~/.config/autoscan/expected_outputs/",
			e.existingExpectedOutputs,
			e.existingExpectedOutputsCursor,
			60,
			8,
			false,
		)
	}

	if e.editingTestCase {
		var b strings.Builder
		if e.editingTestCaseIdx >= 0 {
			b.WriteString(components.RenderHeader("Edit Test Case"))
		} else {
			b.WriteString(components.RenderHeader("Add Test Case"))
		}

		box := components.BoxStyle(80)
		var content strings.Builder

		content.WriteString(e.renderInputRow("Name:          ", e.testCaseInputs.focusedInput == 0, e.testCaseInputs.name, ""))
		content.WriteString(e.renderInputRow(
			"Arguments:     ",
			e.testCaseInputs.focusedInput == 1,
			e.testCaseInputs.args,
			"                   (space-separated)",
		))
		content.WriteString(e.renderInputRow(
			"Stdin:         ",
			e.testCaseInputs.focusedInput == 2,
			e.testCaseInputs.input,
			"                   (use \\n for newlines)",
		))
		content.WriteString(e.renderInputRow(
			"Expected Exit: ",
			e.testCaseInputs.focusedInput == 3,
			e.testCaseInputs.expectedExit,
			"",
		))
		content.WriteString(e.renderValueRow(
			"Expected Output: ",
			e.testCaseInputs.focusedInput == 4,
			e.testCaseInputs.expectedOutputFile,
			"(none)",
		))

		buttonText := "[ Add Test Case ]"
		if e.editingTestCaseIdx >= 0 {
			buttonText = "[ Save Changes ]"
		}
		content.WriteString(components.FocusPrefix(e.testCaseInputs.focusedInput == 5))
		if e.testCaseInputs.focusedInput == 5 {
			content.WriteString(components.SelectedItem.Render(buttonText))
		} else {
			content.WriteString(components.NormalItem.Render(buttonText))
		}

		b.WriteString(box.Render(content.String()))
		b.WriteString("\n\n")
		if e.testCaseInputs.focusedInput == 4 {
			b.WriteString(components.SubtleText.Render("  a add  •  e existing  •  d remove  •  tab/↑↓ navigate  •  esc cancel"))
		} else {
			b.WriteString(components.SubtleText.Render("  tab/↑↓ navigate  •  enter save  •  esc cancel"))
		}

		return b.String()
	}

	if e.browsingForScenarioExpectedOutput {
		return e.renderBrowsePicker(fmt.Sprintf("Select Expected Output for %s", e.scenarioExpectedOutputProcess), "", false)
	}

	if e.showingExistingScenarioExpectedOutput {
		return e.renderExistingPicker(
			fmt.Sprintf("Select Existing Expected Output for %s", e.scenarioExpectedOutputProcess),
			"Expected outputs in ~/.config/autoscan/expected_outputs/",
			"  (no existing expected outputs)",
			e.existingExpectedOutputs,
			e.existingExpectedOutputsCursor,
			70,
			8,
			false,
		)
	}

	if e.editingScenario {
		var b strings.Builder
		if e.editingScenarioIdx >= 0 {
			b.WriteString(components.RenderHeader("Edit Test Scenario"))
		} else {
			b.WriteString(components.RenderHeader("Add Test Scenario"))
		}

		numProcesses := len(e.multiProcessExecs)
		totalFields := 1 + (numProcesses * 4) + 1 // name + (4 fields per process) + save
		saveIdx := totalFields - 1

		box := components.BoxStyle(90)
		var content strings.Builder

		content.WriteString(components.FocusPrefix(e.scenarioInputs.focusedIdx == 0))
		content.WriteString("Scenario Name: ")
		content.WriteString(e.scenarioInputs.name.View())
		content.WriteString("\n\n")

		content.WriteString(components.SubtleText.Render("Configure each process:"))
		content.WriteString("\n\n")

		for i, proc := range e.multiProcessExecs {
			content.WriteString(components.Subtle.Render(fmt.Sprintf("  %s", proc.Name)))
			content.WriteString(components.SubtleText.Render(fmt.Sprintf(" (%s)", proc.SourceFile)))
			content.WriteString("\n")

			argsIdx := 1 + (i * 4)
			stdinIdx := 2 + (i * 4)
			exitIdx := 3 + (i * 4)
			expOutIdx := 4 + (i * 4)

			if input, ok := e.scenarioInputs.processArgs[proc.Name]; ok {
				content.WriteString(e.renderInputRowTight("    Args:     ", e.scenarioInputs.focusedIdx == argsIdx, input))
			}
			if input, ok := e.scenarioInputs.processStdin[proc.Name]; ok {
				content.WriteString(e.renderInputRowTight("    Stdin:    ", e.scenarioInputs.focusedIdx == stdinIdx, input))
			}
			if input, ok := e.scenarioInputs.processExit[proc.Name]; ok {
				content.WriteString(e.renderInputRowTight("    Exit:     ", e.scenarioInputs.focusedIdx == exitIdx, input))
			}
			if expOut, ok := e.scenarioInputs.expectedOutputs[proc.Name]; ok {
				content.WriteString(e.renderValueRowTight("    Expected: ", e.scenarioInputs.focusedIdx == expOutIdx, expOut, "(none)"))
			} else {
				content.WriteString(e.renderValueRowTight("    Expected: ", e.scenarioInputs.focusedIdx == expOutIdx, "", "(none)"))
			}
		}

		buttonText := "[ Add Scenario ]"
		if e.editingScenarioIdx >= 0 {
			buttonText = "[ Save Changes ]"
		}
		content.WriteString(components.FocusPrefix(e.scenarioInputs.focusedIdx == saveIdx))
		if e.scenarioInputs.focusedIdx == saveIdx {
			content.WriteString(components.SelectedItem.Render(buttonText))
		} else {
			content.WriteString(components.NormalItem.Render(buttonText))
		}

		b.WriteString(box.Render(content.String()))
		b.WriteString("\n\n")
		// Check if on expected output field
		isOnExpectedOutput := false
		if e.scenarioInputs.focusedIdx > 0 && e.scenarioInputs.focusedIdx < saveIdx {
			fieldOffset := e.scenarioInputs.focusedIdx - 1
			if fieldOffset%4 == 3 {
				isOnExpectedOutput = true
			}
		}
		if isOnExpectedOutput {
			b.WriteString(components.SubtleText.Render("  a add  •  e existing  •  d remove  •  tab/↑↓ navigate  •  esc cancel"))
		} else {
			b.WriteString(components.SubtleText.Render("  tab/↑↓ navigate  •  enter save  •  esc cancel"))
		}

		return b.String()
	}

	if e.showingExistingLibs {
		return e.renderExistingPicker(
			"Select Existing Library",
			"Libraries bundled in ~/.config/autoscan/libraries/",
			"  (no existing libraries available)",
			e.existingLibs,
			e.existingLibsCursor,
			70,
			8,
			true,
		)
	}

	if e.browsingForLibs {
		return e.renderBrowsePicker("Browse for Library File", "Select a .c or .h file to add", true)
	}

	if e.browsingForTests {
		return e.renderBrowsePicker("Browse for Test File", "Select a test input file to bundle", true)
	}

	if e.showingExistingTests {
		return e.renderExistingPicker(
			"Select Existing Test File",
			"Test files bundled in ~/.config/autoscan/test_files/",
			"  (no existing test files available)",
			e.existingTests,
			e.existingTestsCursor,
			70,
			8,
			true,
		)
	}

	var b strings.Builder

	availableWidth := e.width - 10
	if availableWidth < 100 {
		availableWidth = 100
	}
	colWidth := (availableWidth - 4) / 2
	if colWidth < 45 {
		colWidth = 45
	}
	if colWidth > 80 {
		colWidth = 80
	}
	fullWidth := colWidth*2 + 2

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(components.Primary).
		Padding(0, 2)

	title := "Edit Policy"
	if e.isNew {
		title = "Create New Policy"
	}
	b.WriteString(header.Render(title))
	b.WriteString("\n\n")

	// ═══════════════════════════════════════════════════════════════════════════
	// SECTION 1: GENERAL SETTINGS (name, flags, library files, test files)
	// ═══════════════════════════════════════════════════════════════════════════
	sectionHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(components.Primary)
	b.WriteString(sectionHeader.Render("  GENERAL SETTINGS"))
	b.WriteString("\n")

	smallBoxHeight := 6

	formBox := components.FormBoxStyle().
		Width(colWidth).
		Height(4)

	var nameContent strings.Builder
	nameContent.WriteString(e.renderFieldCompact("Policy Name", e.nameInput.View(), FieldName))
	leftRow1 := formBox.Render(nameContent.String())

	var flagsContent strings.Builder
	flagsContent.WriteString(e.renderFieldCompact("Compiler Flags", e.flagsInput.View(), FieldFlags))
	rightRow1 := formBox.Render(flagsContent.String())

	row1 := lipgloss.JoinHorizontal(lipgloss.Top, leftRow1, "  ", rightRow1)
	b.WriteString(row1)
	b.WriteString("\n")

	libBox := components.FormBoxStyle().
		Width(colWidth).
		Height(smallBoxHeight)
	if e.focusedField == FieldLibraryFiles {
		libBox = libBox.BorderForeground(components.Primary)
	}

	libDisplayItems := make([]string, len(e.libraryFiles))
	for i, f := range e.libraryFiles {
		libDisplayItems[i] = filepath.Base(f)
	}
	libContent := e.renderListSection(
		"Library Files",
		libDisplayItems,
		e.libraryFilesCursor,
		e.focusedField == FieldLibraryFiles,
		smallBoxHeight,
		false,
	)

	tfBox := components.FormBoxStyle().
		Width(colWidth).
		Height(smallBoxHeight)
	if e.focusedField == FieldTestFiles {
		tfBox = tfBox.BorderForeground(components.Primary)
	}

	tfContent := e.renderListSection(
		"Test Files",
		e.testFiles,
		e.testFilesCursor,
		e.focusedField == FieldTestFiles,
		smallBoxHeight,
		false,
	)

	row2 := lipgloss.JoinHorizontal(lipgloss.Top, libBox.Render(libContent), "  ", tfBox.Render(tfContent))
	b.WriteString(row2)
	b.WriteString("\n\n")

	// ═══════════════════════════════════════════════════════════════════════════
	// SECTION 2: EXECUTION MODE TOGGLE (single-process vs multi-process)
	// ═══════════════════════════════════════════════════════════════════════════
	b.WriteString(sectionHeader.Render("  EXECUTION MODE"))
	b.WriteString(components.SubtleText.Render("  e/↵ toggle"))
	b.WriteString("\n")

	modeBox := components.CompactBoxStyle().Width(fullWidth)
	if e.focusedField == FieldMultiProcessToggle {
		modeBox = modeBox.BorderForeground(components.Primary)
	}

	var modeContent strings.Builder
	if !e.multiProcessEnabled {
		modeContent.WriteString(components.SuccessText.Render("●") + " Single Process" + components.SubtleText.Render(" - Compile one source file into one binary"))
		modeContent.WriteString("\n")
		modeContent.WriteString(components.SubtleText.Render("○ Multi-Process - Multiple binaries running in parallel"))
	} else {
		modeContent.WriteString(components.SubtleText.Render("○ Single Process - Compile one source file into one binary"))
		modeContent.WriteString("\n")
		modeContent.WriteString(components.SuccessText.Render("●") + " Multi-Process" + components.SubtleText.Render(" - Multiple binaries running in parallel"))
	}

	b.WriteString(modeBox.Render(modeContent.String()))
	b.WriteString("\n\n")

	// ═══════════════════════════════════════════════════════════════════════════
	// SECTION 3: MODE-SPECIFIC CONFIGURATION
	// Single-process: source file + test cases
	// Multi-process: process list + test scenarios
	// ═══════════════════════════════════════════════════════════════════════════
	if e.multiProcessEnabled {
		b.WriteString(sectionHeader.Render("  MULTI-PROCESS CONFIGURATION"))
	} else {
		b.WriteString(sectionHeader.Render("  SINGLE-PROCESS CONFIGURATION"))
	}
	b.WriteString("\n")

	row2Height := 9

	if !e.multiProcessEnabled {
		srcBox := components.FormBoxStyle().
			Width(colWidth).
			Height(row2Height)
		if e.focusedField == FieldSourceFile {
			srcBox = srcBox.BorderForeground(components.Primary)
		}

		var srcContent strings.Builder
		srcContent.WriteString(e.renderFieldCompact("Source File", e.sourceFileInput.View(), FieldSourceFile))
		srcContent.WriteString(components.SubtleText.Render("  Binary will be named: "))
		sourceFile := strings.TrimSpace(e.sourceFileInput.Value())
		if sourceFile != "" {
			binaryName := sourceFile
			if ext := filepath.Ext(binaryName); ext == ".c" {
				binaryName = binaryName[:len(binaryName)-len(ext)]
			}
			srcContent.WriteString(components.Subtle.Render(binaryName))
		} else {
			srcContent.WriteString(components.SubtleText.Render("(enter source file)"))
		}

		leftCol2 := srcBox.Render(srcContent.String())

		tcBox := components.FormBoxStyle().
			Width(colWidth).
			Height(row2Height)
		if e.focusedField == FieldTestCases {
			tcBox = tcBox.BorderForeground(components.Primary)
		}

		tcDisplayItems := make([]string, len(e.testCases))
		for i, tc := range e.testCases {
			name := tc.Name
			if name == "" {
				name = fmt.Sprintf("Test %d", i+1)
			}
			if len(name) > 30 {
				name = name[:27] + "..."
			}
			tcDisplayItems[i] = name
		}
		tcContent := e.renderListSection(
			"Test Cases",
			tcDisplayItems,
			e.testCasesCursor,
			e.focusedField == FieldTestCases,
			row2Height,
			true,
		)

		rightCol2 := tcBox.Render(tcContent)

		row2 := lipgloss.JoinHorizontal(lipgloss.Top, leftCol2, "  ", rightCol2)
		b.WriteString(row2)
		b.WriteString("\n\n")
	} else {
		mpBox := components.FormBoxStyle().
			Width(colWidth).
			Height(row2Height)
		if e.focusedField == FieldMultiProcess {
			mpBox = mpBox.BorderForeground(components.Primary)
		}

		var mpContent strings.Builder
		if e.focusedField == FieldMultiProcess {
			mpContent.WriteString(components.Highlight.Render("Processes"))
		} else {
			mpContent.WriteString("Processes")
		}
		mpContent.WriteString(components.SubtleText.Render(fmt.Sprintf(" (%d)", len(e.multiProcessExecs))))

		innerHeight := row2Height - 2
		maxItems := innerHeight - 1
		if maxItems < 1 {
			maxItems = 1
		}

		if len(e.multiProcessExecs) == 0 {
			mpContent.WriteString("\n")
			mpContent.WriteString(components.SubtleText.Render("  (none)"))
		} else {
			start, end := e.getScrollWindow(e.multiProcessCursor, len(e.multiProcessExecs), maxItems)
			for i := start; i < end; i++ {
				proc := e.multiProcessExecs[i]
				name := proc.Name
				if len(name) > 25 {
					name = name[:22] + "..."
				}
				mpContent.WriteString("\n")
				if e.focusedField == FieldMultiProcess && i == e.multiProcessCursor {
					mpContent.WriteString("> " + components.SelectedItem.Render(name))
				} else {
					mpContent.WriteString("  " + components.NormalItem.Render(name))
				}
			}
			if len(e.multiProcessExecs) > maxItems {
				mpContent.WriteString("\n")
				mpContent.WriteString(components.SubtleText.Render(fmt.Sprintf("  [%d-%d of %d]", start+1, end, len(e.multiProcessExecs))))
			}
		}

		leftCol2 := mpBox.Render(mpContent.String())

		tsBox := components.FormBoxStyle().
			Width(colWidth).
			Height(row2Height)
		if e.focusedField == FieldMultiProcessTests {
			tsBox = tsBox.BorderForeground(components.Primary)
		}

		tsDisplayItems := make([]string, len(e.testScenarios))
		for i, ts := range e.testScenarios {
			name := ts.Name
			if name == "" {
				name = fmt.Sprintf("Scenario %d", i+1)
			}
			if len(name) > 30 {
				name = name[:27] + "..."
			}
			tsDisplayItems[i] = name
		}
		tsContent := e.renderListSection(
			"Test Scenarios",
			tsDisplayItems,
			e.testScenariosCursor,
			e.focusedField == FieldMultiProcessTests,
			row2Height,
			true,
		)

		rightCol2 := tsBox.Render(tsContent)

		row2 := lipgloss.JoinHorizontal(lipgloss.Top, leftCol2, "  ", rightCol2)
		b.WriteString(row2)
		b.WriteString("\n\n")
	}

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#67E8F9")).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(components.Muted)

	var hints strings.Builder
	switch e.focusedField {
	case FieldMultiProcessToggle:
		hints.WriteString(keyStyle.Render("e") + descStyle.Render(" toggle") + "  ")
	case FieldLibraryFiles:
		hints.WriteString(keyStyle.Render("a") + descStyle.Render(" add") + "  ")
		hints.WriteString(keyStyle.Render("e") + descStyle.Render(" existing") + "  ")
		hints.WriteString(keyStyle.Render("d") + descStyle.Render(" delete") + "  ")
	case FieldTestFiles:
		hints.WriteString(keyStyle.Render("a") + descStyle.Render(" add") + "  ")
		hints.WriteString(keyStyle.Render("e") + descStyle.Render(" existing") + "  ")
		hints.WriteString(keyStyle.Render("d") + descStyle.Render(" delete") + "  ")
	case FieldTestCases:
		hints.WriteString(keyStyle.Render("a") + descStyle.Render(" add") + "  ")
		hints.WriteString(keyStyle.Render("↵") + descStyle.Render(" edit") + "  ")
		hints.WriteString(keyStyle.Render("d") + descStyle.Render(" delete") + "  ")
	case FieldMultiProcess:
		hints.WriteString(keyStyle.Render("a") + descStyle.Render(" add") + "  ")
		hints.WriteString(keyStyle.Render("↵") + descStyle.Render(" edit") + "  ")
		hints.WriteString(keyStyle.Render("d") + descStyle.Render(" delete") + "  ")
	case FieldMultiProcessTests:
		hints.WriteString(keyStyle.Render("a") + descStyle.Render(" add") + "  ")
		hints.WriteString(keyStyle.Render("↵") + descStyle.Render(" edit") + "  ")
		hints.WriteString(keyStyle.Render("d") + descStyle.Render(" delete") + "  ")
	case FieldSave:
		hints.WriteString(keyStyle.Render("↵") + descStyle.Render(" save policy") + "  ")
	case FieldCancel:
		hints.WriteString(keyStyle.Render("↵") + descStyle.Render(" discard changes") + "  ")
	}

	var buttons strings.Builder
	buttons.WriteString("  ")

	if e.focusedField == FieldSave {
		saveBtn := lipgloss.NewStyle().
			Background(lipgloss.Color("#22C55E")).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Padding(0, 2)
		buttons.WriteString(saveBtn.Render("Save"))
	} else {
		saveBtn := lipgloss.NewStyle().
			Background(lipgloss.Color("#374151")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 2)
		buttons.WriteString(saveBtn.Render("Save"))
	}
	buttons.WriteString("  ")

	if e.focusedField == FieldCancel {
		cancelBtn := lipgloss.NewStyle().
			Background(lipgloss.Color("#EF4444")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Padding(0, 2)
		buttons.WriteString(cancelBtn.Render("Cancel"))
	} else {
		cancelBtn := lipgloss.NewStyle().
			Background(lipgloss.Color("#374151")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 2)
		buttons.WriteString(cancelBtn.Render("Cancel"))
	}

	buttons.WriteString("        ")
	buttons.WriteString(hints.String())

	b.WriteString(buttons.String())

	if e.ErrorMsg != "" {
		b.WriteString("\n")
		b.WriteString(components.ErrorStyle.Render("  Error: " + e.ErrorMsg))
	}

	return b.String()
}

func (e *Editor) InSubMode() bool {
	return e.browsingForLibs || e.browsingForTests ||
		e.showingExistingLibs || e.showingExistingTests ||
		e.browsingForExpectedOutput || e.showingExistingExpectedOutput ||
		e.browsingForScenarioExpectedOutput || e.showingExistingScenarioExpectedOutput ||
		e.editingTestCase || e.editingProcess || e.editingScenario
}

func (e *Editor) getScrollWindow(cursor, total, maxVisible int) (start, end int) {
	if total <= maxVisible {
		return 0, total
	}

	halfVisible := maxVisible / 2
	start = cursor - halfVisible
	if start < 0 {
		start = 0
	}
	end = start + maxVisible
	if end > total {
		end = total
		start = end - maxVisible
	}
	return start, end
}

func (e *Editor) renderListSection(title string, items []string, cursor int, focused bool, boxHeight int, editable bool) string {
	innerHeight := boxHeight - 2
	maxItems := innerHeight - 1
	if maxItems < 1 {
		maxItems = 1
	}

	var content strings.Builder

	if focused {
		content.WriteString(components.Highlight.Render(title))
	} else {
		content.WriteString(title)
	}
	content.WriteString(components.SubtleText.Render(fmt.Sprintf(" (%d)", len(items))))

	if len(items) == 0 {
		content.WriteString("\n")
		content.WriteString(components.SubtleText.Render("  (none)"))
	} else {
		start, end := e.getScrollWindow(cursor, len(items), maxItems)
		for i := start; i < end; i++ {
			content.WriteString("\n")
			if focused && i == cursor {
				content.WriteString("> " + components.SelectedItem.Render(items[i]))
			} else {
				content.WriteString("  " + components.NormalItem.Render(items[i]))
			}
		}
		if len(items) > maxItems {
			content.WriteString("\n")
			content.WriteString(components.SubtleText.Render(fmt.Sprintf("  [%d-%d of %d]", start+1, end, len(items))))
		}
	}

	return content.String()
}

func (e *Editor) renderFieldCompact(label, input string, field EditorField) string {
	var b strings.Builder

	if e.focusedField == field {
		b.WriteString(components.Highlight.Render("> " + label + ":"))
	} else {
		b.WriteString(components.Subtle.Render("  " + label + ":"))
	}
	b.WriteString("\n  " + input + "\n")

	return b.String()
}

type (
	SavedMsg struct {
		Path  string
		IsNew bool
	}
	SaveErrorMsg struct {
		Err string
	}
	DeletedMsg struct {
		Name string
	}
)

func DeletePolicy(p *policy.Policy) tea.Cmd {
	return func() tea.Msg {
		if err := os.Remove(p.FilePath); err != nil {
			return DeleteErrorMsg{Err: err}
		}
		return DeletedMsg{Name: p.Name}
	}
}
