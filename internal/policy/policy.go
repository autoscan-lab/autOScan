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

	// RequiredFiles lists source files that must be present (e.g., ["S0.c", "S1.c"])
	RequiredFiles []string `yaml:"required_files"`

	// Report configures export options
	Report ReportConfig `yaml:"report"`

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

	// Output is the output binary name
	Output string `yaml:"output"`
}

// ReportConfig controls export options.
type ReportConfig struct {
	// Export configures which formats to export
	Export ExportConfig `yaml:"export"`
}

// ExportConfig controls which export formats are enabled.
type ExportConfig struct {
	Markdown bool `yaml:"markdown"`
	JSON     bool `yaml:"json"`
	CSV      bool `yaml:"csv"`
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
func (p *Policy) BuildGCCArgs(sourceFiles []string, outputPath string) []string {
	args := []string{}

	// Add all flags (compiler + linker combined)
	args = append(args, p.Compile.Flags...)

	// Add source files
	args = append(args, sourceFiles...)

	// Add output
	args = append(args, "-o", outputPath)

	return args
}

// LoadGlobalBanned loads banned functions from a global config file.
func LoadGlobalBanned(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var functions []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		functions = append(functions, line)
	}
	return functions, nil
}
