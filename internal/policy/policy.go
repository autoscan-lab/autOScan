// Package policy handles loading and parsing of YAML policy files.
package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felipetrejos/autoscan/internal/config"
	"gopkg.in/yaml.v3"
)

// Policy defines the grading rules for a lab.
type Policy struct {
	// Name is the human-readable name of the policy (e.g., "Lab 03 - Threads")
	Name string `yaml:"name"`

	// Root is the default root folder to scan (optional, default ".")
	Root string `yaml:"root"`

	// Discover configures how submissions are discovered
	Discover DiscoverConfig `yaml:"discover"`

	// Compile configures gcc compilation
	Compile CompileConfig `yaml:"compile"`

	// Run configures execution/testing of compiled binaries
	Run RunConfig `yaml:"run"`

	// RequiredFiles lists source files that must be present (e.g., ["S0.c", "S1.c"])
	RequiredFiles []string `yaml:"required_files"`

	// LibraryFiles lists additional source files to compile with each submission
	// These are typically instructor-provided library files (e.g., ["lib/utils.c"])
	LibraryFiles []string `yaml:"library_files"`

	// TestFiles lists input files bundled for testing (e.g., ["input.txt", "data.bin"])
	// These are copied to ~/.config/autoscan/test_files/ and can be referenced in args
	TestFiles []string `yaml:"test_files,omitempty"`

	// FilePath is the path to the policy file (set after loading)
	FilePath string `yaml:"-"`

	// BannedFunctions loaded from global config (not from YAML)
	BannedFunctions []string `yaml:"-"`
}

// DiscoverConfig controls submission discovery.
type DiscoverConfig struct {
	// LeafSubmission when true, treats leaf folders as submissions
	LeafSubmission bool `yaml:"leaf_submission"`

	// MinCFiles is the minimum number of .c files required (default 1)
	MinCFiles int `yaml:"min_c_files"`
}

// CompileConfig controls gcc compilation.
type CompileConfig struct {
	// GCC is the path to gcc (default "gcc")
	GCC string `yaml:"gcc"`

	// Flags are all compiler/linker flags (e.g., ["-Wall", "-Wextra", "-lpthread"])
	Flags []string `yaml:"flags"`

	// SourceFile is the main source file to compile (e.g., "S5.c")
	// When set, only this file is compiled and the binary is named after it (S5.c -> S5)
	// When empty, all .c files in the submission are compiled together
	SourceFile string `yaml:"source_file,omitempty"`

	// Output is the output binary name (legacy, use SourceFile instead)
	// Only used when SourceFile is empty
	Output string `yaml:"output,omitempty"`
}

// RunConfig controls execution/testing of compiled binaries.
type RunConfig struct {
	// TestCases are predefined test scenarios
	TestCases []TestCase `yaml:"test_cases"`

	// MultiProcess enables running multiple source files in parallel
	// Useful for labs with producer/consumer or synchronization patterns
	MultiProcess *MultiProcessConfig `yaml:"multi_process,omitempty"`
}

// MultiProcessConfig defines how to run multiple binaries in parallel.
type MultiProcessConfig struct {
	// Enabled activates multi-process mode
	Enabled bool `yaml:"enabled"`

	// Executables defines the separate binaries to run
	// Each maps a source file to its execution config
	Executables []ProcessConfig `yaml:"executables"`

	// TestScenarios defines multiple test configurations for the processes
	// Each scenario can have different args/inputs per process
	TestScenarios []MultiProcessScenario `yaml:"test_scenarios,omitempty"`
}

// ProcessConfig defines how to run a single process in multi-process mode.
type ProcessConfig struct {
	// Name is a display name (e.g., "Producer", "Consumer")
	Name string `yaml:"name"`

	// SourceFile is the .c file that produces this binary
	SourceFile string `yaml:"source_file"`

	// Args are the DEFAULT command-line arguments for this process
	// These are used when running without a specific test scenario
	Args []string `yaml:"args,omitempty"`

	// Input is stdin for this process (default)
	Input string `yaml:"input,omitempty"`

	// StartDelay in milliseconds before starting (for staggered starts)
	StartDelayMs int `yaml:"start_delay_ms,omitempty"`
}

// MultiProcessScenario defines a test configuration for all processes.
type MultiProcessScenario struct {
	// Name is a display name for this scenario
	Name string `yaml:"name"`

	// ProcessArgs maps process name -> arguments for that process
	ProcessArgs map[string][]string `yaml:"process_args,omitempty"`

	// ProcessInputs maps process name -> stdin input for that process
	ProcessInputs map[string]string `yaml:"process_inputs,omitempty"`

	// ExpectedExits maps process name -> expected exit code
	ExpectedExits map[string]int `yaml:"expected_exits,omitempty"`
}

// TestCase defines a single test scenario for running a submission.
type TestCase struct {
	// Name is a human-readable name for this test case
	Name string `yaml:"name"`

	// Args are command-line arguments to pass
	Args []string `yaml:"args"`

	// Input is stdin input to provide
	Input string `yaml:"input"`

	// ExpectedExit is the expected exit code (0 for success)
	ExpectedExit *int `yaml:"expected_exit"`
}

// Load reads and parses a policy file from the given path.
func Load(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading policy file: %w", err)
	}

	var p Policy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing policy YAML: %w", err)
	}

	// Set defaults
	p.FilePath = path
	if p.Root == "" {
		p.Root = "."
	}
	if p.Discover.MinCFiles == 0 {
		p.Discover.MinCFiles = 1
	}
	if p.Compile.GCC == "" {
		p.Compile.GCC = "gcc"
	}
	if p.Compile.Output == "" {
		p.Compile.Output = "a.out"
	}

	return &p, nil
}

// Discover finds all policy files in a directory.
func Discover(dir string) ([]*Policy, error) {
	var policies []*Policy

	// Load global banned functions from config directory
	bannedFile, _ := config.BannedFile()
	bannedFuncs, _ := LoadGlobalBanned(bannedFile)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No policies directory is fine
		}
		return nil, fmt.Errorf("reading policies directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		path := filepath.Join(dir, name)
		p, err := Load(path)
		if err != nil {
			// Log warning but continue with other policies
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", name, err)
			continue
		}

		// Attach global banned functions
		p.BannedFunctions = bannedFuncs

		policies = append(policies, p)
	}

	return policies, nil
}

// BannedSet returns the banned functions as a set for fast lookup.
func (p *Policy) BannedSet() map[string]struct{} {
	set := make(map[string]struct{}, len(p.BannedFunctions))
	for _, fn := range p.BannedFunctions {
		set[fn] = struct{}{}
	}
	return set
}

// BuildGCCArgs constructs the gcc command arguments for a list of source files.
// libraryFiles are additional source files (with full paths) to compile with the submission.
func (p *Policy) BuildGCCArgs(sourceFiles []string, libraryFiles []string, outputPath string) []string {
	args := []string{}

	// Separate compiler flags from linker flags
	// Compiler flags: -Wall, -Wextra, -g, -O2, etc.
	// Linker flags: -lpthread, -lm, -lrt, etc. (start with -l)
	var compilerFlags []string
	var linkerFlags []string

	for _, flag := range p.Compile.Flags {
		if strings.HasPrefix(flag, "-l") {
			// Linker flag (library) - must come after object files
			linkerFlags = append(linkerFlags, flag)
		} else {
			// Compiler flag - comes before source files
			compilerFlags = append(compilerFlags, flag)
		}
	}

	// Add compiler flags first
	args = append(args, compilerFlags...)

	// Add source files (.c files - will be compiled)
	args = append(args, sourceFiles...)

	// Add library files (instructor-provided code)
	// Only add .c and .o files - .h files are included via #include
	for _, libFile := range libraryFiles {
		if strings.HasSuffix(libFile, ".c") || strings.HasSuffix(libFile, ".o") {
			args = append(args, libFile)
		}
		// .h files are not passed to gcc - they're found via #include
		// They just need to be in the libraries directory (added via -I flag if needed)
	}

	// Add linker flags AFTER object files (required by gcc)
	args = append(args, linkerFlags...)

	// Add output
	args = append(args, "-o", outputPath)

	return args
}

// LoadGlobalBanned loads banned functions from a YAML config file.
func LoadGlobalBanned(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var config struct {
		Banned []string `yaml:"banned"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing banned.yaml: %w", err)
	}

	return config.Banned, nil
}
