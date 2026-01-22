package app

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/felipetrejos/autoscan/internal/tui"
)

func installPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin", "autoscan")
}

// Run initializes and starts the application
func Run() error {
	// Check for install/uninstall commands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			return install()
		case "uninstall":
			return uninstall()
		}
	}

	// Parse command line flags
	var (
		policyPath = flag.String("policy", "", "Path to policy YAML file")
		rootPath   = flag.String("root", ".", "Root folder containing submissions")
	)
	flag.Parse()

	// Create TUI config
	cfg := tui.Config{
		PolicyPath: *policyPath,
		Root:       *rootPath,
	}

	return tui.Start(cfg)
}

func install() error {
	dest := installPath()

	// Get current executable path
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("resolving symlinks: %w", err)
	}

	// Check if already installed
	if exe == dest {
		fmt.Println("autoscan is already installed.")
		return nil
	}

	// Ensure ~/.local/bin exists
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Copy binary
	src, err := os.Open(exe)
	if err != nil {
		return fmt.Errorf("opening source: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("creating destination: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copying binary: %w", err)
	}

	fmt.Printf("✓ Installed to %s\n", dest)
	fmt.Println()

	// Check if ~/.local/bin is in PATH
	pathEnv := os.Getenv("PATH")
	home, _ := os.UserHomeDir()
	localBin := filepath.Join(home, ".local", "bin")
	if !pathContains(pathEnv, localBin) {
		fmt.Println("Add to your ~/.zshrc:")
		fmt.Printf("  export PATH=\"$HOME/.local/bin:$PATH\"\n")
		fmt.Println()
		fmt.Println("Then run: source ~/.zshrc")
	} else {
		fmt.Println("You can now run 'autoscan' from anywhere.")
	}
	return nil
}

func pathContains(pathEnv, dir string) bool {
	for _, p := range filepath.SplitList(pathEnv) {
		if p == dir {
			return true
		}
	}
	return false
}

func uninstall() error {
	dest := installPath()
	if err := os.Remove(dest); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("autoscan is not installed.")
			return nil
		}
		return err
	}
	fmt.Printf("✓ Removed %s\n", dest)
	return nil
}
