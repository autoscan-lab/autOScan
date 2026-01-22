package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipetrejos/autoscan/internal/tui/styles"
)

// FolderBrowser allows navigating and selecting folders
type FolderBrowser struct {
	currentPath  string
	entries      []string // folder names in current directory
	cursor       int
	scrollOffset int
	visibleRows  int
	selected     string
	err          string
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
	fb.err = ""

	entries, err := os.ReadDir(fb.currentPath)
	if err != nil {
		fb.err = err.Error()
		return
	}

	// Filter to directories only, skip hidden
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			fb.entries = append(fb.entries, e.Name())
		}
	}

	sort.Slice(fb.entries, func(i, j int) bool {
		return strings.ToLower(fb.entries[i]) < strings.ToLower(fb.entries[j])
	})
}

// Update handles keyboard input
func (fb *FolderBrowser) Update(msg tea.KeyMsg) (selected bool, cmd tea.Cmd) {
	totalItems := len(fb.entries) + 2 // +2 for ".." and "[Select This Folder]"

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

// CurrentPath returns the current path being browsed
func (fb *FolderBrowser) CurrentPath() string {
	return fb.currentPath
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

	// Build items list
	items := []string{"[Select This Folder]", ".."}
	items = append(items, fb.entries...)

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

		name := items[i]
		icon := ""
		if i == 0 {
			icon = "* " // Select current
		} else if i == 1 {
			icon = "^ " // Parent
		} else {
			icon = "/ " // Folder
		}

		nameStyle := styles.NormalItem
		if i == fb.cursor {
			nameStyle = styles.SelectedItem
		}

		line := cursor + icon + nameStyle.Render(name)
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

// SetVisibleRows sets how many rows to display
func (fb *FolderBrowser) SetVisibleRows(rows int) {
	fb.visibleRows = rows
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
	fb.loadEntries()
}
