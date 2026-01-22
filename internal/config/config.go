// Package config handles configuration directory setup and embedded defaults.
package config

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed defaults/*
var defaultsFS embed.FS

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

// BannedFile returns the banned.txt file path
func BannedFile() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "banned.txt"), nil
}

// Init ensures the config directory exists with default files.
// Called on app startup. Only creates files if they don't exist.
func Init() error {
	configDir, err := Dir()
	if err != nil {
		return fmt.Errorf("getting config dir: %w", err)
	}

	// Check if already initialized
	if _, err := os.Stat(configDir); err == nil {
		return nil // Already exists
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

		// Calculate destination path (strip "defaults/" prefix)
		relPath, _ := filepath.Rel("defaults", path)
		destPath := filepath.Join(configDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
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
