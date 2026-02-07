package app

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/felitrejos/autoscan/internal/tui"
)

// Run initializes and starts the application
func Run() error {
	// Auto-install to ~/.local/bin if not already there
	autoInstall()

	// Parse command line flags
	rootPath := flag.String("root", ".", "Root folder containing submissions")
	flag.Parse()

	// Create TUI config
	cfg := tui.Config{
		Root: *rootPath,
	}

	return tui.Start(cfg)
}

func shellRC() string {
	if runtime.GOOS == "darwin" {
		return "~/.zshrc"
	}
	// Linux
	return "~/.bashrc"
}

func autoInstall() {
	home, _ := os.UserHomeDir()
	localBin := filepath.Join(home, ".local", "bin")
	dest := filepath.Join(localBin, "autoscan")

	exe, err := os.Executable()
	if err != nil {
		return
	}
	exe, _ = filepath.EvalSymlinks(exe)

	// Skip if already running from ~/.local/bin
	if exe == dest {
		if !inPath(localBin) {
			fmt.Printf("Add to %s to run 'autoscan' from anywhere:\n", shellRC())
			fmt.Println("  export PATH=\"$HOME/.local/bin:$PATH\"")
			fmt.Println()
		}
		return
	}

	// Create ~/.local/bin if needed
	if err := os.MkdirAll(localBin, 0755); err != nil {
		return
	}

	// Copy binary to ~/.local/bin
	src, err := os.Open(exe)
	if err != nil {
		return
	}
	defer src.Close()

	dst, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return
	}

	fmt.Printf("Installed to %s\n", dest)
	if !inPath(localBin) {
		fmt.Println()
		fmt.Printf("Add to %s to run 'autoscan' from anywhere:\n", shellRC())
		fmt.Println("  export PATH=\"$HOME/.local/bin:$PATH\"")
	}
	fmt.Println()
}

func inPath(dir string) bool {
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p == dir {
			return true
		}
	}
	return false
}
