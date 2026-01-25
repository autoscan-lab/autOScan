package config

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed defaults/*
var defaultsFS embed.FS

// Settings holds the global settings
type Settings struct {
	ShortNames bool `yaml:"short_names"`
	KeepBinaries bool `yaml:"keep_binaries"`
	MaxWorkers int `yaml:"max_workers"`
}

// DefaultSettings returns the default settings
func DefaultSettings() Settings {
	return Settings{
		ShortNames:   true,
		KeepBinaries: false,
		MaxWorkers:   0, // 0 = use all CPUs
	}
}

// Dir returns the config directory path
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "autoscan"), nil
}

// PoliciesDir returns the policies directory path
func PoliciesDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "policies"), nil
}

// BannedFile returns the banned.yaml file path
func BannedFile() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "banned.yaml"), nil
}

// LibrariesDir returns the libraries directory path
func LibrariesDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "libraries"), nil
}

// EnsureLibrariesDir creates the libraries directory
func EnsureLibrariesDir() (string, error) {
	libDir, err := LibrariesDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(libDir, 0755); err != nil {
		return "", err
	}
	return libDir, nil
}

// TestFilesDir returns the test_files directory path
func TestFilesDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "test_files"), nil
}

// EnsureTestFilesDir creates the test_files directory
func EnsureTestFilesDir() (string, error) {
	testDir, err := TestFilesDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return "", err
	}
	return testDir, nil
}

// SettingsFile returns the settings.yaml path
func SettingsFile() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.yaml"), nil
}

// LoadSettings loads the settings from settings.yaml
func LoadSettings() (Settings, error) {
	settingsFile, err := SettingsFile()
	if err != nil {
		return DefaultSettings(), err
	}

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultSettings(), nil
		}
		return DefaultSettings(), err
	}

	var settings Settings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return DefaultSettings(), err
	}

	return settings, nil
}

// SaveSettings saves settings to settings.yaml
func SaveSettings(s Settings) error {
	settingsFile, err := SettingsFile()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(&s)
	if err != nil {
		return err
	}

	return os.WriteFile(settingsFile, []byte(string(data)), 0644)
}

// Init ensures the config directory exists with default files
func Init() error {
	configDir, err := Dir()
	if err != nil {
		return fmt.Errorf("getting config dir: %w", err)
	}

	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	// Copy embedded defaults
	err = fs.WalkDir(defaultsFS, "defaults", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel("defaults", path)
		destPath := filepath.Join(configDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		if _, err := os.Stat(destPath); err == nil {
			return nil
		}

		data, err := defaultsFS.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, 0644)
	})

	if err != nil {
		return fmt.Errorf("copying defaults: %w", err)
	}

	return nil
}
