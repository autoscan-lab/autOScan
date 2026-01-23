// Package config handles configuration directory setup and embedded defaults.
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

// Settings holds global application settings
type Settings struct {
	// ShortNames truncates student folder names at the first underscore
	ShortNames bool `yaml:"short_names"`
	// KeepBinaries controls whether compiled binaries are deleted after grading
	KeepBinaries bool `yaml:"keep_binaries"`
}

// DefaultSettings returns default settings
func DefaultSettings() Settings {
	return Settings{
		ShortNames: true,
		KeepBinaries: false,
	}
}

// Dir returns the config directory path (~/.config/autoscan)
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

// SettingsFile returns the settings.yaml file path
func SettingsFile() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.yaml"), nil
}

// LoadSettings loads settings from settings.yaml
func LoadSettings() (Settings, error) {
	settingsFile, err := SettingsFile()
	if err != nil {
		return DefaultSettings(), err
	}

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		// Return defaults if file doesn't exist
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

	header := "# autOScan Settings\n\n"
	return os.WriteFile(settingsFile, []byte(header+string(data)), 0644)
}

// Init ensures the config directory exists with default files.
// Called on app startup. Creates missing files if they don't exist.
func Init() error {
	configDir, err := Dir()
	if err != nil {
		return fmt.Errorf("getting config dir: %w", err)
	}

	// Create config directory if needed
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	// Copy embedded defaults (only if files don't exist)
	err = fs.WalkDir(defaultsFS, "defaults", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path (strip "defaults/" prefix)
		relPath, _ := filepath.Rel("defaults", path)
		destPath := filepath.Join(configDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// Skip if file already exists
		if _, err := os.Stat(destPath); err == nil {
			return nil
		}

		// Read embedded file
		data, err := defaultsFS.ReadFile(path)
		if err != nil {
			return err
		}

		// Write to config dir
		return os.WriteFile(destPath, data, 0644)
	})

	if err != nil {
		return fmt.Errorf("copying defaults: %w", err)
	}

	return nil
}
