package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/feli05/autoscan/internal/tui/styles"
)

// FolderBrowser allows navigating and selecting folders or files
type FolderBrowser struct {
	currentPath  string
	entries      []string // folder/file names in current directory
	isDir        []bool   // true if entry is a directory
	cursor       int
	scrollOffset int
	visibleRows  int
	selected     string
	err          string
	fileMode     bool     // when true, shows and allows selecting .c/.h files
	fileExts     []string // allowed file extensions in file mode
}

// NewFolderBrowser creates a new folder browser starting at the given path
func NewFolderBrowser(startPath string) FolderBrowser {
	if startPath == "" {
		startPath = "."
	}

	absPath, err := filepath.Abs(startPath)
	if err != nil {
		absPath = startPath
	}

	fb := FolderBrowser{
		currentPath: absPath,
		visibleRows: 12,
	}
	fb.loadEntries()
	return fb
}

func (fb *FolderBrowser) loadEntries() {
	fb.entries = nil
	fb.isDir = nil
	fb.err = ""

	entries, err := os.ReadDir(fb.currentPath)
	if err != nil {
		fb.err = err.Error()
		return
	}

	// Collect directories first
	var dirs []string
	var files []string

	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue // Skip hidden
		}

		if e.IsDir() {
			dirs = append(dirs, name)
		} else if fb.fileMode {
			// In file mode, also include matching files
			for _, ext := range fb.fileExts {
				if strings.HasSuffix(strings.ToLower(name), ext) {
					files = append(files, name)
					break
				}
			}
		}
	}

	// Sort directories and files separately
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i]) < strings.ToLower(dirs[j])
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i]) < strings.ToLower(files[j])
	})

	// Add directories first, then files
	for _, d := range dirs {
		fb.entries = append(fb.entries, d)
		fb.isDir = append(fb.isDir, true)
	}
	for _, f := range files {
		fb.entries = append(fb.entries, f)
		fb.isDir = append(fb.isDir, false)
	}
}

// Update handles keyboard input
func (fb *FolderBrowser) Update(msg tea.KeyMsg) (selected bool, cmd tea.Cmd) {
	// Calculate number of fixed items at top
	fixedItems := 1 // ".." only in file mode
	if !fb.fileMode {
		fixedItems = 2 // "[Select This Folder]" and ".."
	}
	totalItems := len(fb.entries) + fixedItems

	switch msg.String() {
	case "j", "down":
		if fb.cursor < totalItems-1 {
			fb.cursor++
			if fb.cursor >= fb.scrollOffset+fb.visibleRows {
				fb.scrollOffset++
			}
		}
	case "k", "up":
		if fb.cursor > 0 {
			fb.cursor--
			if fb.cursor < fb.scrollOffset {
				fb.scrollOffset--
			}
		}
	case "enter":
		if fb.fileMode {
			// File mode: 0 = "..", rest = entries
			if fb.cursor == 0 {
				// ".." - go up
				fb.currentPath = filepath.Dir(fb.currentPath)
				fb.cursor = 0
				fb.scrollOffset = 0
				fb.loadEntries()
			} else {
				idx := fb.cursor - 1
				if idx < len(fb.entries) {
					if fb.isDir[idx] {
						// Enter subfolder
						fb.currentPath = filepath.Join(fb.currentPath, fb.entries[idx])
						fb.cursor = 0
						fb.scrollOffset = 0
						fb.loadEntries()
					} else {
						// Select file
						fb.selected = filepath.Join(fb.currentPath, fb.entries[idx])
						return true, nil
					}
				}
			}
		} else {
			// Folder mode: 0 = "[Select This Folder]", 1 = "..", rest = entries
			if fb.cursor == 0 {
				// "[Select This Folder]" - select current directory
				fb.selected = fb.currentPath
				return true, nil
			} else if fb.cursor == 1 {
				// ".." - go up
				fb.currentPath = filepath.Dir(fb.currentPath)
				fb.cursor = 0
				fb.scrollOffset = 0
				fb.loadEntries()
			} else {
				// Enter subfolder
				idx := fb.cursor - 2
				if idx < len(fb.entries) {
					fb.currentPath = filepath.Join(fb.currentPath, fb.entries[idx])
					fb.cursor = 0
					fb.scrollOffset = 0
					fb.loadEntries()
				}
			}
		}
	case "backspace", "h", "left":
		// Go up one directory
		fb.currentPath = filepath.Dir(fb.currentPath)
		fb.cursor = 0
		fb.scrollOffset = 0
		fb.loadEntries()
	}
	return false, nil
}

// Selected returns the selected path
func (fb *FolderBrowser) Selected() string {
	return fb.selected
}

// View renders the folder browser
func (fb *FolderBrowser) View() string {
	var b strings.Builder

	// Current path header
	b.WriteString(styles.Subtle.Render(fb.currentPath))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 50))
	b.WriteString("\n")

	if fb.err != "" {
		b.WriteString(styles.ErrorStyle.Render("Error: " + fb.err))
		b.WriteString("\n")
		return b.String()
	}

	// Build items list based on mode
	type item struct {
		name  string
		icon  string
		isDir bool
	}
	var items []item

	if fb.fileMode {
		// File mode: ".." + entries
		items = append(items, item{name: "..", icon: "^ ", isDir: true})
	} else {
		// Folder mode: "[Select This Folder]" + ".." + entries
		items = append(items, item{name: "[Select This Folder]", icon: "* ", isDir: true})
		items = append(items, item{name: "..", icon: "^ ", isDir: true})
	}

	for i, name := range fb.entries {
		icon := "/ " // Folder
		if !fb.isDir[i] {
			icon = "# " // File
		}
		items = append(items, item{name: name, icon: icon, isDir: fb.isDir[i]})
	}

	if len(items) == 0 {
		b.WriteString(styles.Subtle.Render("  (empty directory)"))
		b.WriteString("\n")
		return b.String()
	}

	endIdx := fb.scrollOffset + fb.visibleRows
	if endIdx > len(items) {
		endIdx = len(items)
	}

	for i := fb.scrollOffset; i < endIdx; i++ {
		cursor := "  "
		if i == fb.cursor {
			cursor = "> "
		}

		it := items[i]
		nameStyle := styles.NormalItem
		if i == fb.cursor {
			nameStyle = styles.SelectedItem
		}

		// Use different color for files vs folders
		iconStyle := styles.Subtle
		if !it.isDir {
			iconStyle = styles.SuccessText
		}

		line := cursor + iconStyle.Render(it.icon) + nameStyle.Render(it.name)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(items) > fb.visibleRows {
		b.WriteString(styles.Subtle.Render(fmt.Sprintf("\n  %d-%d of %d",
			fb.scrollOffset+1, endIdx, len(items))))
	}

	return b.String()
}

// Reset resets the browser to start path
func (fb *FolderBrowser) Reset(startPath string) {
	if startPath == "" {
		startPath = "."
	}
	absPath, _ := filepath.Abs(startPath)
	fb.currentPath = absPath
	fb.cursor = 0
	fb.scrollOffset = 0
	fb.selected = ""
	fb.err = ""
	fb.fileMode = false
	fb.fileExts = nil
	fb.loadEntries()
}

// SetFileMode enables file selection mode with specified extensions
func (fb *FolderBrowser) SetFileMode(enabled bool) {
	fb.fileMode = enabled
	if enabled {
		fb.fileExts = []string{".c", ".h"} // Default to C source files
	} else {
		fb.fileExts = nil
	}
	fb.loadEntries()
}

// SetFileExtensions sets allowed file extensions for file mode
func (fb *FolderBrowser) SetFileExtensions(exts []string) {
	fb.fileExts = exts
	if fb.fileMode {
		fb.loadEntries()
	}
}
