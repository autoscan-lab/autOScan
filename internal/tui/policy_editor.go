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

type PolicyEditorField int

const (
	FieldName PolicyEditorField = iota
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

type PolicyEditor struct {
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

	folderBrowser    FolderBrowser
	browsingForLibs  bool
	browsingForTests bool
	browsingStartDir string

	showingExistingLibs bool
	existingLibs        []string
	existingLibsCursor  int

	showingExistingTests bool
	existingTests        []string
	existingTestsCursor  int

	testCases          []policy.TestCase
	testCasesCursor    int
	editingTestCase    bool
	editingTestCaseIdx int
	testCaseInputs     struct {
		name         textinput.Model
		args         textinput.Model
		input        textinput.Model
		expectedExit textinput.Model
		focusedInput int
	}

	runTimeout          string
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

	testScenarios          []policy.MultiProcessScenario
	testScenariosCursor    int
	editingScenario        bool
	editingScenarioIdx     int
	scenarioInputs         struct {
		name         textinput.Model
		processArgs  map[string]textinput.Model
		processStdin map[string]textinput.Model
		processExit  map[string]textinput.Model
		focusedIdx   int
	}

	focusedField PolicyEditorField
	errorMsg     string
}

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

	// Process config inputs for multi-process mode
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

	// Scenario name input
	scenarioNameInput := textinput.New()
	scenarioNameInput.Placeholder = "Test Scenario Name"
	scenarioNameInput.CharLimit = 50
	scenarioNameInput.Width = 40

	pe := PolicyEditor{
		isNew:           true,
		nameInput:       nameInput,
		flagsInput:      flagsInput,
		sourceFileInput: sourceFileInput,
		libraryFiles:    []string{},
		folderBrowser:   NewFolderBrowser(cwd),
		browsingStartDir: cwd,
		focusedField:    FieldName,
		testCases:       []policy.TestCase{},
		runTimeout:      "5s",
		multiProcessExecs:  []policy.ProcessConfig{},
		testScenarios:      []policy.MultiProcessScenario{},
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

	return pe
}

func (e *PolicyEditor) LoadPolicy(p *policy.Policy) {
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
	e.runTimeout = p.Run.Timeout
	if e.runTimeout == "" {
		e.runTimeout = "5s"
	}

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

func (e *PolicyEditor) SetWidth(w int) {
	e.width = w
}

func (e *PolicyEditor) Reset() {
	e.isNew = true
	e.filePath = ""
	e.focusedField = FieldName
	e.errorMsg = ""
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

	// Reset test cases
	e.testCases = []policy.TestCase{}
	e.testCasesCursor = 0
	e.editingTestCase = false
	e.editingTestCaseIdx = -1
	e.runTimeout = "5s"
	e.resetTestCaseInputs()

	// Reset multi-process
	e.multiProcessEnabled = false
	e.multiProcessExecs = []policy.ProcessConfig{}
	e.multiProcessCursor = 0
	e.editingProcess = false
	e.editingProcessIdx = -1
	e.resetProcessInputs()

	// Reset test scenarios
	e.testScenarios = []policy.MultiProcessScenario{}
	e.testScenariosCursor = 0
	e.editingScenario = false
	e.editingScenarioIdx = -1
	e.resetScenarioInputs()
}

func (e *PolicyEditor) resetTestCaseInputs() {
	e.testCaseInputs.name.SetValue("")
	e.testCaseInputs.args.SetValue("")
	e.testCaseInputs.input.SetValue("")
	e.testCaseInputs.expectedExit.SetValue("0")
	e.testCaseInputs.focusedInput = 0
	e.testCaseInputs.name.Focus()
	e.testCaseInputs.args.Blur()
	e.testCaseInputs.input.Blur()
	e.testCaseInputs.expectedExit.Blur()
}

func (e *PolicyEditor) resetProcessInputs() {
	e.processInputs.name.SetValue("")
	e.processInputs.sourceFile.SetValue("")
	e.processInputs.args.SetValue("")
	e.processInputs.delayMs.SetValue("0")
	e.processInputs.focusedIdx = 0
	e.processInputs.name.Focus()
	e.processInputs.sourceFile.Blur()
	e.processInputs.args.Blur()
	e.processInputs.delayMs.Blur()
}

func (e *PolicyEditor) resetScenarioInputs() {
	e.scenarioInputs.name.SetValue("")
	e.scenarioInputs.name.Focus()
	e.scenarioInputs.focusedIdx = 0
	e.scenarioInputs.processArgs = make(map[string]textinput.Model)
	e.scenarioInputs.processStdin = make(map[string]textinput.Model)
	e.scenarioInputs.processExit = make(map[string]textinput.Model)
}

// initScenarioProcessInputs initializes input fields for each process in a scenario
func (e *PolicyEditor) initScenarioProcessInputs() {
	e.scenarioInputs.processArgs = make(map[string]textinput.Model)
	e.scenarioInputs.processStdin = make(map[string]textinput.Model)
	e.scenarioInputs.processExit = make(map[string]textinput.Model)

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

func (e *PolicyEditor) blurAllScenarioInputs() {
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

func (e *PolicyEditor) focusCurrentScenarioInput() {
	numProcesses := len(e.multiProcessExecs)
	totalFields := 1 + (numProcesses * 3) + 1

	if e.scenarioInputs.focusedIdx == 0 {
		e.scenarioInputs.name.Focus()
	} else if e.scenarioInputs.focusedIdx < totalFields-1 {
		fieldOffset := e.scenarioInputs.focusedIdx - 1
		procIdx := fieldOffset / 3
		fieldType := fieldOffset % 3

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
			}
		}
	}
}

// loadExistingTestFiles loads list of files from the test_files directory
func (e *PolicyEditor) loadExistingTestFiles() {
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
		// Check if not already in policy
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

// loadExistingLibraries loads list of files from the libraries directory
func (e *PolicyEditor) loadExistingLibraries() {
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
			// Check if not already in policy
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

func (e *PolicyEditor) Update(msg tea.Msg) tea.Cmd {
	if e.showingExistingLibs {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				e.showingExistingLibs = false
				return nil
			case "j", "down":
				if e.existingLibsCursor < len(e.existingLibs)-1 {
					e.existingLibsCursor++
				}
				return nil
			case "k", "up":
				if e.existingLibsCursor > 0 {
					e.existingLibsCursor--
				}
				return nil
			case "enter":
				if e.existingLibsCursor < len(e.existingLibs) {
					// Add selected library to policy
					e.libraryFiles = append(e.libraryFiles, e.existingLibs[e.existingLibsCursor])
				}
				e.showingExistingLibs = false
				return nil
			}
		}
		return nil
	}

	// If browsing for library files, delegate to folder browser
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
				// Get selected path and copy to libraries directory
				selectedPath := e.folderBrowser.Selected()

				// Check if it's a .c, .h, or .o file
				if strings.HasSuffix(selectedPath, ".c") || strings.HasSuffix(selectedPath, ".h") || strings.HasSuffix(selectedPath, ".o") {
					// Get the filename
					filename := filepath.Base(selectedPath)

					// Check if not already in list
					alreadyExists := false
					for _, f := range e.libraryFiles {
						if f == filename {
							alreadyExists = true
							break
						}
					}

					if !alreadyExists {
						// Copy file to libraries directory
						libDir, err := config.EnsureLibrariesDir()
						if err == nil {
							destPath := filepath.Join(libDir, filename)
							// Read source file
							data, err := os.ReadFile(selectedPath)
							if err == nil {
								// Write to libraries directory
								if err := os.WriteFile(destPath, data, 0644); err == nil {
									// Store just the filename (file is now bundled)
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

	// If browsing for test files, delegate to folder browser
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

				// Check if not already in list
				alreadyExists := false
				for _, f := range e.testFiles {
					if f == filename {
						alreadyExists = true
						break
					}
				}

				if !alreadyExists {
					// Copy file to test_files directory
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

	// If showing existing test files picker
	if e.showingExistingTests {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				e.showingExistingTests = false
				return nil
			case "j", "down":
				if e.existingTestsCursor < len(e.existingTests)-1 {
					e.existingTestsCursor++
				}
				return nil
			case "k", "up":
				if e.existingTestsCursor > 0 {
					e.existingTestsCursor--
				}
				return nil
			case "enter":
				if e.existingTestsCursor < len(e.existingTests) {
					e.testFiles = append(e.testFiles, e.existingTests[e.existingTestsCursor])
				}
				e.showingExistingTests = false
				return nil
			}
		}
		return nil
	}

	// If editing a test case
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
				e.testCaseInputs.focusedInput = (e.testCaseInputs.focusedInput + 1) % 5
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
				e.testCaseInputs.focusedInput = (e.testCaseInputs.focusedInput + 4) % 5
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
			case "enter":
				if e.testCaseInputs.focusedInput == 4 {
					// Save button - create or update the test case
					tc := policy.TestCase{
						Name:  e.testCaseInputs.name.Value(),
						Input: e.testCaseInputs.input.Value(),
					}
					if tc.Name == "" {
						tc.Name = fmt.Sprintf("Test %d", len(e.testCases)+1)
					}
					// Parse args
					if args := e.testCaseInputs.args.Value(); args != "" {
						tc.Args = strings.Fields(args)
					}
					// Parse expected exit
					if exitStr := e.testCaseInputs.expectedExit.Value(); exitStr != "" {
						var exitCode int
						if _, err := fmt.Sscanf(exitStr, "%d", &exitCode); err == nil {
							tc.ExpectedExit = &exitCode
						}
					}

					// Update existing or add new
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

			// Update focused input
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

	// If editing a process config
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
					// Save button - create the process config
					proc := policy.ProcessConfig{
						Name:       e.processInputs.name.Value(),
						SourceFile: e.processInputs.sourceFile.Value(),
					}
					if proc.Name == "" {
						proc.Name = fmt.Sprintf("Process %d", len(e.multiProcessExecs)+1)
					}
					if proc.SourceFile == "" {
						proc.SourceFile = "main.c"
					}
					// Parse args
					if args := e.processInputs.args.Value(); args != "" {
						proc.Args = strings.Fields(args)
					}
					// Parse delay
					if delayStr := e.processInputs.delayMs.Value(); delayStr != "" {
						var delay int
						if _, err := fmt.Sscanf(delayStr, "%d", &delay); err == nil {
							proc.StartDelayMs = delay
						}
					}

					// Update existing or add new
					if e.editingProcessIdx >= 0 && e.editingProcessIdx < len(e.multiProcessExecs) {
						e.multiProcessExecs[e.editingProcessIdx] = proc
					} else {
						e.multiProcessExecs = append(e.multiProcessExecs, proc)
						e.multiProcessEnabled = true // Enable when adding first process
					}
					e.editingProcess = false
					e.editingProcessIdx = -1
					e.resetProcessInputs()
					return nil
				}
			}

			// Update focused input
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

	// If editing a scenario
	if e.editingScenario {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			numProcesses := len(e.multiProcessExecs)
			// Total fields: name (0) + 3 per process + save button
			totalFields := 1 + (numProcesses * 3) + 1

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
			case "enter":
				saveIdx := totalFields - 1
				if e.scenarioInputs.focusedIdx == saveIdx {
					// Save button - create the scenario
					scenario := policy.MultiProcessScenario{
						Name:          e.scenarioInputs.name.Value(),
						ProcessArgs:   make(map[string][]string),
						ProcessInputs: make(map[string]string),
						ExpectedExits: make(map[string]int),
					}
					if scenario.Name == "" {
						scenario.Name = fmt.Sprintf("Scenario %d", len(e.testScenarios)+1)
					}

					// Collect values for each process
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
					}

					// Update existing or add new
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

			// Update focused input
			var cmd tea.Cmd
			if e.scenarioInputs.focusedIdx == 0 {
				e.scenarioInputs.name, cmd = e.scenarioInputs.name.Update(msg)
			} else if e.scenarioInputs.focusedIdx < totalFields-1 {
				// Process field
				fieldOffset := e.scenarioInputs.focusedIdx - 1
				procIdx := fieldOffset / 3
				fieldType := fieldOffset % 3

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

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle multi-process field specially
		if e.focusedField == FieldMultiProcess {
			switch msg.String() {
			case "a":
				// Add new process
				e.editingProcess = true
				e.editingProcessIdx = -1
				e.resetProcessInputs()
				return nil
			case "enter":
				// Edit selected process
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
				// Delete selected process
				if len(e.multiProcessExecs) > 0 && e.multiProcessCursor < len(e.multiProcessExecs) {
					e.multiProcessExecs = append(e.multiProcessExecs[:e.multiProcessCursor], e.multiProcessExecs[e.multiProcessCursor+1:]...)
					if e.multiProcessCursor >= len(e.multiProcessExecs) && e.multiProcessCursor > 0 {
						e.multiProcessCursor--
					}
					// Disable if no more processes
					if len(e.multiProcessExecs) == 0 {
						e.multiProcessEnabled = false
					}
				}
				return nil
			case "e":
				// Toggle enabled
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

		// Handle multi-process test scenarios field
		if e.focusedField == FieldMultiProcessTests {
			switch msg.String() {
			case "a":
				// Add new scenario (only if processes are defined)
				if len(e.multiProcessExecs) > 0 {
					e.editingScenario = true
					e.editingScenarioIdx = -1
					e.resetScenarioInputs()
					e.initScenarioProcessInputs()
					e.scenarioInputs.name.Focus()
				}
				return nil
			case "enter":
				// Edit selected scenario
				if len(e.testScenarios) > 0 && e.testScenariosCursor < len(e.testScenarios) {
					scenario := e.testScenarios[e.testScenariosCursor]
					e.editingScenario = true
					e.editingScenarioIdx = e.testScenariosCursor
					e.initScenarioProcessInputs()
					e.scenarioInputs.name.SetValue(scenario.Name)
					// Load process-specific values
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
					}
					e.scenarioInputs.focusedIdx = 0
					e.scenarioInputs.name.Focus()
				}
				return nil
			case "d", "backspace":
				// Delete selected scenario
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

		// Handle multi-process toggle
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

		// Handle test cases field specially
		if e.focusedField == FieldTestCases {
			switch msg.String() {
			case "a":
				// Add new test case
				e.editingTestCase = true
				e.editingTestCaseIdx = -1
				e.resetTestCaseInputs()
				return nil
			case "enter":
				// Edit selected test case
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
					e.testCaseInputs.focusedInput = 0
					e.testCaseInputs.name.Focus()
				}
				return nil
			case "d", "backspace":
				// Delete selected test case
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

		// Handle library files field specially
		if e.focusedField == FieldLibraryFiles {
			switch msg.String() {
			case "a":
				// Add new library file via browser (from filesystem)
				cwd, _ := os.Getwd()
				e.folderBrowser.Reset(cwd)
				e.folderBrowser.SetFileMode(true)
				e.folderBrowser.SetFileExtensions([]string{".c", ".h", ".o"})
				e.browsingForLibs = true
				return nil
			case "e":
				// Add from existing bundled libraries
				e.loadExistingLibraries()
				e.existingLibsCursor = 0
				if len(e.existingLibs) > 0 {
					e.showingExistingLibs = true
				}
				return nil
			case "d", "backspace":
				// Delete selected library file
				if len(e.libraryFiles) > 0 && e.libraryFilesCursor < len(e.libraryFiles) {
					e.libraryFiles = append(e.libraryFiles[:e.libraryFilesCursor], e.libraryFiles[e.libraryFilesCursor+1:]...)
					if e.libraryFilesCursor >= len(e.libraryFiles) && e.libraryFilesCursor > 0 {
						e.libraryFilesCursor--
					}
				}
				return nil
			case "j", "down":
				// Navigate within library files list, or next field if at end/empty
				if len(e.libraryFiles) > 0 && e.libraryFilesCursor < len(e.libraryFiles)-1 {
					e.libraryFilesCursor++
				} else {
					e.nextField()
				}
				return nil
			case "k", "up":
				// Navigate within library files list, or prev field if at start/empty
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

		// Handle test files field specially
		if e.focusedField == FieldTestFiles {
			switch msg.String() {
			case "a":
				// Add new test file via browser
				cwd, _ := os.Getwd()
				e.folderBrowser.Reset(cwd)
				e.folderBrowser.SetFileMode(true)
				e.folderBrowser.SetFileExtensions([]string{".txt", ".bin", ".dat", ".in", ".out"})
				e.browsingForTests = true
				return nil
			case "e":
				// Add from existing bundled test files
				e.loadExistingTestFiles()
				e.existingTestsCursor = 0
				if len(e.existingTests) > 0 {
					e.showingExistingTests = true
				}
				return nil
			case "d", "backspace":
				// Delete selected test file
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

	// Update the focused input
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

func (e *PolicyEditor) nextField() {
	e.blurAll()
	e.focusedField++
	// Skip mode-specific fields based on current mode
	e.focusedField = e.adjustFieldForMode(e.focusedField, true)
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
		// Skip mode-specific fields based on current mode
		e.focusedField = e.adjustFieldForMode(e.focusedField, false)
	}
	e.focusCurrent()
}

// adjustFieldForMode skips fields that don't apply to current mode
func (e *PolicyEditor) adjustFieldForMode(field PolicyEditorField, forward bool) PolicyEditorField {
	// If multi-process is enabled, skip single-process fields
	if e.multiProcessEnabled {
		if field == FieldSourceFile || field == FieldTestCases {
			if forward {
				return FieldMultiProcess
			}
			return FieldMultiProcessToggle
		}
	} else {
		// If multi-process is disabled, skip multi-process fields
		if field == FieldMultiProcess || field == FieldMultiProcessTests {
			if forward {
				return FieldSave
			}
			return FieldTestCases
		}
	}
	return field
}

func (e *PolicyEditor) blurAll() {
	e.nameInput.Blur()
	e.flagsInput.Blur()
	e.sourceFileInput.Blur()
}

func (e *PolicyEditor) focusCurrent() {
	switch e.focusedField {
	case FieldName:
		e.nameInput.Focus()
	case FieldFlags:
		e.flagsInput.Focus()
	case FieldSourceFile:
		e.sourceFileInput.Focus()
	}
}

func (e *PolicyEditor) save() tea.Cmd {
	return func() tea.Msg {
		// Validate
		name := strings.TrimSpace(e.nameInput.Value())
		if name == "" {
			return policySaveErrorMsg{err: "Policy name is required"}
		}

		// Parse flags
		flagsStr := strings.TrimSpace(e.flagsInput.Value())
		var flags []string
		if flagsStr != "" {
			flags = strings.Fields(flagsStr)
		}

		// Get source file for single-process mode
		sourceFile := strings.TrimSpace(e.sourceFileInput.Value())

		// Build policy struct for YAML
		type TestCaseYAML struct {
			Name         string   `yaml:"name,omitempty"`
			Args         []string `yaml:"args,omitempty"`
			Input        string   `yaml:"input,omitempty"`
			ExpectedExit *int     `yaml:"expected_exit,omitempty"`
		}

		p := struct {
			Name     string `yaml:"name"`
			Root     string `yaml:"root"`
			Discover struct {
				LeafSubmission bool `yaml:"leaf_submission"`
				MinCFiles      int  `yaml:"min_c_files"`
			} `yaml:"discover"`
			Compile struct {
				GCC        string   `yaml:"gcc"`
				Flags      []string `yaml:"flags"`
				SourceFile string   `yaml:"source_file,omitempty"`
			} `yaml:"compile"`
			Run struct {
				Timeout      string         `yaml:"timeout,omitempty"`
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
						Name          string              `yaml:"name"`
						ProcessArgs   map[string][]string `yaml:"process_args,omitempty"`
						ProcessInputs map[string]string   `yaml:"process_inputs,omitempty"`
						ExpectedExits map[string]int      `yaml:"expected_exits,omitempty"`
					} `yaml:"test_scenarios,omitempty"`
				} `yaml:"multi_process,omitempty"`
			} `yaml:"run,omitempty"`
			LibraryFiles []string `yaml:"library_files,omitempty"`
			TestFiles    []string `yaml:"test_files,omitempty"`
			Report       struct {
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
		p.Compile.SourceFile = sourceFile
		p.LibraryFiles = e.libraryFiles
		p.TestFiles = e.testFiles
		p.Report.Export.Markdown = true

		// Add run config if there are test cases or multi-process
		if len(e.testCases) > 0 || e.runTimeout != "" || len(e.multiProcessExecs) > 0 {
			p.Run.Timeout = e.runTimeout
			for _, tc := range e.testCases {
				p.Run.TestCases = append(p.Run.TestCases, TestCaseYAML{
					Name:         tc.Name,
					Args:         tc.Args,
					Input:        tc.Input,
					ExpectedExit: tc.ExpectedExit,
				})
			}

			// Add multi-process config
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
						Name          string              `yaml:"name"`
						ProcessArgs   map[string][]string `yaml:"process_args,omitempty"`
						ProcessInputs map[string]string   `yaml:"process_inputs,omitempty"`
						ExpectedExits map[string]int      `yaml:"expected_exits,omitempty"`
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
				// Add test scenarios
				for _, scenario := range e.testScenarios {
					p.Run.MultiProcess.TestScenarios = append(p.Run.MultiProcess.TestScenarios, struct {
						Name          string              `yaml:"name"`
						ProcessArgs   map[string][]string `yaml:"process_args,omitempty"`
						ProcessInputs map[string]string   `yaml:"process_inputs,omitempty"`
						ExpectedExits map[string]int      `yaml:"expected_exits,omitempty"`
					}{
						Name:          scenario.Name,
						ProcessArgs:   scenario.ProcessArgs,
						ProcessInputs: scenario.ProcessInputs,
						ExpectedExits: scenario.ExpectedExits,
					})
				}
			}
		}

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

func (e *PolicyEditor) View() string {
	if e.editingProcess {
		var b strings.Builder
		if e.editingProcessIdx >= 0 {
			b.WriteString(styles.HeaderStyle.Render("Edit Process"))
		} else {
			b.WriteString(styles.HeaderStyle.Render("Add Process"))
		}
		b.WriteString("\n\n")

		box := styles.BoxStyle(80)
		var content strings.Builder

		// Use plain text prefix for consistent alignment
		cursor := func(focused bool) string {
			if focused {
				return "> "
			}
			return "  "
		}

		// Name field
		content.WriteString(cursor(e.processInputs.focusedIdx == 0))
		content.WriteString("Name:      ")
		content.WriteString(e.processInputs.name.View())
		content.WriteString("\n\n")

		// Source file field
		content.WriteString(cursor(e.processInputs.focusedIdx == 1))
		content.WriteString("Source:    ")
		content.WriteString(e.processInputs.sourceFile.View())
		content.WriteString("\n")
		content.WriteString(styles.SubtleText.Render("             (binary = filename without .c)"))
		content.WriteString("\n\n")

		// Args field
		content.WriteString(cursor(e.processInputs.focusedIdx == 2))
		content.WriteString("Arguments: ")
		content.WriteString(e.processInputs.args.View())
		content.WriteString("\n\n")

		// Delay field
		content.WriteString(cursor(e.processInputs.focusedIdx == 3))
		content.WriteString("Delay (ms):")
		content.WriteString(e.processInputs.delayMs.View())
		content.WriteString("\n\n")

		// Save button
		buttonText := "[ Add Process ]"
		if e.editingProcessIdx >= 0 {
			buttonText = "[ Save Changes ]"
		}
		content.WriteString(cursor(e.processInputs.focusedIdx == 4))
		if e.processInputs.focusedIdx == 4 {
			content.WriteString(styles.SelectedItem.Render(buttonText))
		} else {
			content.WriteString(styles.NormalItem.Render(buttonText))
		}

		b.WriteString(box.Render(content.String()))
		b.WriteString("\n\n")
		b.WriteString(styles.SubtleText.Render("  tab/↑↓ navigate  •  enter save  •  esc cancel"))

		return b.String()
	}

	// If editing a test case, show the test case editor
	if e.editingTestCase {
		var b strings.Builder
		if e.editingTestCaseIdx >= 0 {
			b.WriteString(styles.HeaderStyle.Render("Edit Test Case"))
		} else {
			b.WriteString(styles.HeaderStyle.Render("Add Test Case"))
		}
		b.WriteString("\n\n")

		box := styles.BoxStyle(80)
		var content strings.Builder

		// Use plain text prefix for consistent alignment
		cursor := func(focused bool) string {
			if focused {
				return "> "
			}
			return "  "
		}

		// Name field
		content.WriteString(cursor(e.testCaseInputs.focusedInput == 0))
		content.WriteString("Name:          ")
		content.WriteString(e.testCaseInputs.name.View())
		content.WriteString("\n\n")

		// Args field
		content.WriteString(cursor(e.testCaseInputs.focusedInput == 1))
		content.WriteString("Arguments:     ")
		content.WriteString(e.testCaseInputs.args.View())
		content.WriteString("\n")
		content.WriteString(styles.SubtleText.Render("                   (space-separated)"))
		content.WriteString("\n\n")

		// Input field
		content.WriteString(cursor(e.testCaseInputs.focusedInput == 2))
		content.WriteString("Stdin:         ")
		content.WriteString(e.testCaseInputs.input.View())
		content.WriteString("\n")
		content.WriteString(styles.SubtleText.Render("                   (use \\n for newlines)"))
		content.WriteString("\n\n")

		// Expected exit field
		content.WriteString(cursor(e.testCaseInputs.focusedInput == 3))
		content.WriteString("Expected Exit: ")
		content.WriteString(e.testCaseInputs.expectedExit.View())
		content.WriteString("\n\n")

		// Save button
		buttonText := "[ Add Test Case ]"
		if e.editingTestCaseIdx >= 0 {
			buttonText = "[ Save Changes ]"
		}
		content.WriteString(cursor(e.testCaseInputs.focusedInput == 4))
		if e.testCaseInputs.focusedInput == 4 {
			content.WriteString(styles.SelectedItem.Render(buttonText))
		} else {
			content.WriteString(styles.NormalItem.Render(buttonText))
		}

		b.WriteString(box.Render(content.String()))
		b.WriteString("\n\n")
		b.WriteString(styles.SubtleText.Render("  tab/↑↓ navigate  •  enter add  •  esc cancel"))

		return b.String()
	}

	// If editing a test scenario (multi-process)
	if e.editingScenario {
		var b strings.Builder
		if e.editingScenarioIdx >= 0 {
			b.WriteString(styles.HeaderStyle.Render("Edit Test Scenario"))
		} else {
			b.WriteString(styles.HeaderStyle.Render("Add Test Scenario"))
		}
		b.WriteString("\n\n")

		numProcesses := len(e.multiProcessExecs)
		totalFields := 1 + (numProcesses * 3) + 1
		saveIdx := totalFields - 1

		box := styles.BoxStyle(90)
		var content strings.Builder

		cursor := func(focused bool) string {
			if focused {
				return "> "
			}
			return "  "
		}

		// Scenario name
		content.WriteString(cursor(e.scenarioInputs.focusedIdx == 0))
		content.WriteString("Scenario Name: ")
		content.WriteString(e.scenarioInputs.name.View())
		content.WriteString("\n\n")

		// Per-process fields
		content.WriteString(styles.SubtleText.Render("Configure each process:"))
		content.WriteString("\n\n")

		for i, proc := range e.multiProcessExecs {
			content.WriteString(styles.Subtle.Render(fmt.Sprintf("  %s", proc.Name)))
			content.WriteString(styles.SubtleText.Render(fmt.Sprintf(" (%s)", proc.SourceFile)))
			content.WriteString("\n")

			argsIdx := 1 + (i * 3)
			stdinIdx := 2 + (i * 3)
			exitIdx := 3 + (i * 3)

			// Args field
			content.WriteString(cursor(e.scenarioInputs.focusedIdx == argsIdx))
			content.WriteString("    Args:  ")
			if input, ok := e.scenarioInputs.processArgs[proc.Name]; ok {
				content.WriteString(input.View())
			}
			content.WriteString("\n")

			// Stdin field
			content.WriteString(cursor(e.scenarioInputs.focusedIdx == stdinIdx))
			content.WriteString("    Stdin: ")
			if input, ok := e.scenarioInputs.processStdin[proc.Name]; ok {
				content.WriteString(input.View())
			}
			content.WriteString("\n")

			// Expected exit field
			content.WriteString(cursor(e.scenarioInputs.focusedIdx == exitIdx))
			content.WriteString("    Exit:  ")
			if input, ok := e.scenarioInputs.processExit[proc.Name]; ok {
				content.WriteString(input.View())
			}
			content.WriteString("\n\n")
		}

		// Save button
		buttonText := "[ Add Scenario ]"
		if e.editingScenarioIdx >= 0 {
			buttonText = "[ Save Changes ]"
		}
		content.WriteString(cursor(e.scenarioInputs.focusedIdx == saveIdx))
		if e.scenarioInputs.focusedIdx == saveIdx {
			content.WriteString(styles.SelectedItem.Render(buttonText))
		} else {
			content.WriteString(styles.NormalItem.Render(buttonText))
		}

		b.WriteString(box.Render(content.String()))
		b.WriteString("\n\n")
		b.WriteString(styles.SubtleText.Render("  tab/↑↓ navigate  •  enter save  •  esc cancel"))

		return b.String()
	}

	// If showing existing libraries picker
	if e.showingExistingLibs {
		var b strings.Builder
		b.WriteString(styles.HeaderStyle.Render("Select Existing Library"))
		b.WriteString("\n\n")

		box := styles.BoxStyle(70)
		var content strings.Builder
		content.WriteString(styles.SubtleText.Render("Libraries bundled in ~/.config/autoscan/libraries/"))
		content.WriteString("\n\n")

		if len(e.existingLibs) == 0 {
			content.WriteString(styles.SubtleText.Render("  (no existing libraries available)\n"))
		} else {
			maxVisible := 8
			start, end := e.getScrollWindow(e.existingLibsCursor, len(e.existingLibs), maxVisible)
			for i := start; i < end; i++ {
				lib := e.existingLibs[i]
				if i == e.existingLibsCursor {
					content.WriteString("> " + styles.SelectedItem.Render(lib) + "\n")
				} else {
					content.WriteString("  " + styles.NormalItem.Render(lib) + "\n")
				}
			}
			if len(e.existingLibs) > maxVisible {
				content.WriteString(styles.SubtleText.Render(fmt.Sprintf("\n  [%d-%d of %d]\n", start+1, end, len(e.existingLibs))))
			}
		}

		b.WriteString(box.Render(content.String()))
		b.WriteString("\n\n")
		b.WriteString(styles.SubtleText.Render("  ↑↓ navigate  •  enter select  •  esc cancel"))

		return b.String()
	}

	// If browsing for library files, show the folder browser
	if e.browsingForLibs {
		var b strings.Builder
		b.WriteString(styles.HeaderStyle.Render("Browse for Library File"))
		b.WriteString("\n\n")

		box := styles.BoxStyle(60)
		var content strings.Builder
		content.WriteString(styles.SubtleText.Render("Select a .c or .h file to add"))
		content.WriteString("\n\n")
		content.WriteString(e.folderBrowser.View())
		b.WriteString(box.Render(content.String()))

		b.WriteString("\n\n")
		b.WriteString(styles.SubtleText.Render("  enter select  •  esc cancel"))

		return b.String()
	}

	// If browsing for test files, show the folder browser
	if e.browsingForTests {
		var b strings.Builder
		b.WriteString(styles.HeaderStyle.Render("Browse for Test File"))
		b.WriteString("\n\n")

		box := styles.BoxStyle(60)
		var content strings.Builder
		content.WriteString(styles.SubtleText.Render("Select a test input file to bundle"))
		content.WriteString("\n\n")
		content.WriteString(e.folderBrowser.View())
		b.WriteString(box.Render(content.String()))

		b.WriteString("\n\n")
		b.WriteString(styles.SubtleText.Render("  enter select  •  esc cancel"))

		return b.String()
	}

	// If showing existing test files picker
	if e.showingExistingTests {
		var b strings.Builder
		b.WriteString(styles.HeaderStyle.Render("Select Existing Test File"))
		b.WriteString("\n\n")

		box := styles.BoxStyle(70)
		var content strings.Builder
		content.WriteString(styles.SubtleText.Render("Test files bundled in ~/.config/autoscan/test_files/"))
		content.WriteString("\n\n")

		if len(e.existingTests) == 0 {
			content.WriteString(styles.SubtleText.Render("  (no existing test files available)\n"))
		} else {
			maxVisible := 8
			start, end := e.getScrollWindow(e.existingTestsCursor, len(e.existingTests), maxVisible)
			for i := start; i < end; i++ {
				tf := e.existingTests[i]
				if i == e.existingTestsCursor {
					content.WriteString("> " + styles.SelectedItem.Render(tf) + "\n")
				} else {
					content.WriteString("  " + styles.NormalItem.Render(tf) + "\n")
				}
			}
			if len(e.existingTests) > maxVisible {
				content.WriteString(styles.SubtleText.Render(fmt.Sprintf("\n  [%d-%d of %d]\n", start+1, end, len(e.existingTests))))
			}
		}

		b.WriteString(box.Render(content.String()))
		b.WriteString("\n\n")
		b.WriteString(styles.SubtleText.Render("  ↑↓ navigate  •  enter select  •  esc cancel"))

		return b.String()
	}

	var b strings.Builder

	// Column widths - responsive based on terminal width
	availableWidth := e.width - 10 // Leave some margin
	if availableWidth < 100 {
		availableWidth = 100
	}
	colWidth := (availableWidth - 4) / 2 // Two columns with gap
	if colWidth < 45 {
		colWidth = 45
	}
	if colWidth > 80 {
		colWidth = 80 // Don't make columns too wide
	}
	fullWidth := colWidth*2 + 2

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		Padding(0, 2)

	title := "Edit Policy"
	if e.isNew {
		title = "Create New Policy"
	}
	b.WriteString(header.Render(title))
	b.WriteString("\n\n")

	// ─────────────────────────────────────────────────────────────────────────
	// SECTION 1: GENERAL SETTINGS
	// ─────────────────────────────────────────────────────────────────────────

	sectionHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary)
	b.WriteString(sectionHeader.Render("  GENERAL SETTINGS"))
	b.WriteString("\n")

	smallBoxHeight := 6

	// ROW 1: Policy Name (left) | Compiler Flags (right)
	formBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(0, 1).
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

	// ROW 2: Library Files (left) | Test Files (right)
	libBorder := styles.Muted
	if e.focusedField == FieldLibraryFiles {
		libBorder = styles.Primary
	}
	libBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(libBorder).
		Padding(0, 1).
		Width(colWidth).
		Height(smallBoxHeight)

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

	tfBorder := styles.Muted
	if e.focusedField == FieldTestFiles {
		tfBorder = styles.Primary
	}
	tfBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tfBorder).
		Padding(0, 1).
		Width(colWidth).
		Height(smallBoxHeight)

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

	// ─────────────────────────────────────────────────────────────────────────
	// SECTION 2: EXECUTION MODE
	// ─────────────────────────────────────────────────────────────────────────

	b.WriteString(sectionHeader.Render("  EXECUTION MODE"))
	b.WriteString(styles.SubtleText.Render("  e/↵ toggle"))
	b.WriteString("\n")

	// Mode toggle box (full width)
	modeBorder := styles.Muted
	if e.focusedField == FieldMultiProcessToggle {
		modeBorder = styles.Primary
	}
	modeBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(modeBorder).
		Padding(0, 2).
		Width(fullWidth)

	var modeContent strings.Builder
	if !e.multiProcessEnabled {
		modeContent.WriteString(styles.SuccessText.Render("● Single Process"))
		modeContent.WriteString(styles.SubtleText.Render(" - Compile one source file into one binary"))
	} else {
		modeContent.WriteString(styles.SubtleText.Render("○ Single Process"))
		modeContent.WriteString(styles.SubtleText.Render(" - Compile one source file into one binary"))
	}
	modeContent.WriteString("    ")
	if e.multiProcessEnabled {
		modeContent.WriteString(styles.SuccessText.Render("● Multi-Process"))
		modeContent.WriteString(styles.SubtleText.Render(" - Multiple binaries running in parallel"))
	} else {
		modeContent.WriteString(styles.SubtleText.Render("○ Multi-Process"))
		modeContent.WriteString(styles.SubtleText.Render(" - Multiple binaries running in parallel"))
	}

	b.WriteString(modeBox.Render(modeContent.String()))
	b.WriteString("\n\n")

	// ─────────────────────────────────────────────────────────────────────────
	// SECTION 3: MODE-SPECIFIC CONFIGURATION
	// ─────────────────────────────────────────────────────────────────────────

	if e.multiProcessEnabled {
		b.WriteString(sectionHeader.Render("  MULTI-PROCESS CONFIGURATION"))
	} else {
		b.WriteString(sectionHeader.Render("  SINGLE-PROCESS CONFIGURATION"))
	}
	b.WriteString("\n")

	row2Height := 9

	if !e.multiProcessEnabled {
		// ─── SINGLE-PROCESS MODE ───
		// LEFT: Source file input (match height of Test Cases box)
		srcBorder := styles.Muted
		if e.focusedField == FieldSourceFile {
			srcBorder = styles.Primary
		}
		srcBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(srcBorder).
			Padding(0, 1).
			Width(colWidth).
			Height(row2Height)

		var srcContent strings.Builder
		srcContent.WriteString(e.renderFieldCompact("Source File", e.sourceFileInput.View(), FieldSourceFile))
		srcContent.WriteString(styles.SubtleText.Render("  Binary will be named: "))
		sourceFile := strings.TrimSpace(e.sourceFileInput.Value())
		if sourceFile != "" {
			binaryName := sourceFile
			if ext := filepath.Ext(binaryName); ext == ".c" {
				binaryName = binaryName[:len(binaryName)-len(ext)]
			}
			srcContent.WriteString(styles.Subtle.Render(binaryName))
		} else {
			srcContent.WriteString(styles.SubtleText.Render("(enter source file)"))
		}

		leftCol2 := srcBox.Render(srcContent.String())

		// RIGHT: Test Cases
		tcBorder := styles.Muted
		if e.focusedField == FieldTestCases {
			tcBorder = styles.Primary
		}
		tcBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tcBorder).
			Padding(0, 1).
			Width(colWidth).
			Height(row2Height)

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
		// ─── MULTI-PROCESS MODE ───
		// LEFT: Processes list
		mpBorder := styles.Muted
		if e.focusedField == FieldMultiProcess {
			mpBorder = styles.Primary
		}
		mpBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mpBorder).
			Padding(0, 1).
			Width(colWidth).
			Height(row2Height)

		var mpContent strings.Builder
		if e.focusedField == FieldMultiProcess {
			mpContent.WriteString(styles.Highlight.Render("Processes"))
		} else {
			mpContent.WriteString("Processes")
		}
		mpContent.WriteString(styles.SubtleText.Render(fmt.Sprintf(" (%d)", len(e.multiProcessExecs))))

		innerHeight := row2Height - 2
		maxItems := innerHeight - 1
		if maxItems < 1 {
			maxItems = 1
		}

		if len(e.multiProcessExecs) == 0 {
			mpContent.WriteString("\n")
			mpContent.WriteString(styles.SubtleText.Render("  (none)"))
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
					mpContent.WriteString("> " + styles.SelectedItem.Render(name))
				} else {
					mpContent.WriteString("  " + styles.NormalItem.Render(name))
				}
			}
			if len(e.multiProcessExecs) > maxItems {
				mpContent.WriteString("\n")
				mpContent.WriteString(styles.SubtleText.Render(fmt.Sprintf("  [%d-%d of %d]", start+1, end, len(e.multiProcessExecs))))
			}
		}

		leftCol2 := mpBox.Render(mpContent.String())

		// RIGHT: Test Scenarios
		tsBorder := styles.Muted
		if e.focusedField == FieldMultiProcessTests {
			tsBorder = styles.Primary
		}
		tsBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tsBorder).
			Padding(0, 1).
			Width(colWidth).
			Height(row2Height)

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

	// Footer row: Buttons on left, context hints on right
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#67E8F9")).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(styles.Muted)

	// Build context-specific hints
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

	// Buttons
	var buttons strings.Builder
	buttons.WriteString("  ")

	// Save button
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

	// Cancel button
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

	// Add spacing then hints
	buttons.WriteString("        ")
	buttons.WriteString(hints.String())

	b.WriteString(buttons.String())

	// Error message
	if e.errorMsg != "" {
		b.WriteString("\n")
		b.WriteString(styles.ErrorStyle.Render("  Error: " + e.errorMsg))
	}

	return b.String()
}

func (e *PolicyEditor) InSubMode() bool {
	return e.browsingForLibs || e.browsingForTests ||
		e.showingExistingLibs || e.showingExistingTests ||
		e.editingTestCase || e.editingProcess || e.editingScenario
}

// getScrollWindow calculates the visible window for a scrollable list
func (e *PolicyEditor) getScrollWindow(cursor, total, maxVisible int) (start, end int) {
	if total <= maxVisible {
		return 0, total
	}

	// Keep cursor roughly centered
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

// renderListSection renders a list section with title and items (no hints - global hints used)
func (e *PolicyEditor) renderListSection(title string, items []string, cursor int, focused bool, boxHeight int, editable bool) string {
	// Inner height = boxHeight - 2 (border)
	innerHeight := boxHeight - 2
	maxItems := innerHeight - 1 // Reserve 1 for title
	if maxItems < 1 {
		maxItems = 1
	}

	var content strings.Builder

	// Title - editable sections get a subtle indicator
	if focused {
		content.WriteString(styles.Highlight.Render(title))
	} else {
		content.WriteString(title)
	}
	content.WriteString(styles.SubtleText.Render(fmt.Sprintf(" (%d)", len(items))))

	if len(items) == 0 {
		content.WriteString("\n")
		content.WriteString(styles.SubtleText.Render("  (none)"))
	} else {
		start, end := e.getScrollWindow(cursor, len(items), maxItems)
		for i := start; i < end; i++ {
			content.WriteString("\n")
			if focused && i == cursor {
				content.WriteString("> " + styles.SelectedItem.Render(items[i]))
			} else {
				content.WriteString("  " + styles.NormalItem.Render(items[i]))
			}
		}
		if len(items) > maxItems {
			content.WriteString("\n")
			content.WriteString(styles.SubtleText.Render(fmt.Sprintf("  [%d-%d of %d]", start+1, end, len(items))))
		}
	}

	return content.String()
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

func (e *PolicyEditor) renderFieldCompact(label, input string, field PolicyEditorField) string {
	var b strings.Builder

	if e.focusedField == field {
		b.WriteString(styles.Highlight.Render("> " + label + ":"))
	} else {
		b.WriteString(styles.Subtle.Render("  " + label + ":"))
	}
	b.WriteString("\n  " + input + "\n")

	return b.String()
}

func (e *PolicyEditor) renderFieldCompactWithHint(label, input string, field PolicyEditorField) string {
	var b strings.Builder

	// Split label and hint if present
	labelParts := strings.SplitN(label, " (", 2)
	mainLabel := labelParts[0]
	var hint string
	if len(labelParts) > 1 {
		hint = "(" + labelParts[1] // Restore the "("
	}

	if e.focusedField == field {
		b.WriteString(styles.Highlight.Render("> " + mainLabel + ":"))
	} else {
		b.WriteString(styles.Subtle.Render("  " + mainLabel + ":"))
	}
	
	// Add hint in subtle text if present
	if hint != "" {
		b.WriteString(" " + styles.SubtleText.Render(hint))
	}
	
	b.WriteString("\n  " + input + "\n")

	return b.String()
}

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

func DeletePolicy(p *policy.Policy) tea.Cmd {
	return func() tea.Msg {
		if err := os.Remove(p.FilePath); err != nil {
			return errorMsg(err)
		}
		return policyDeletedMsg{name: p.Name}
	}
}
