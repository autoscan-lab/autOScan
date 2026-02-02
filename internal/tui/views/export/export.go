package export

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/feli05/autoscan/internal/domain"
	exportpkg "github.com/feli05/autoscan/internal/export"
	"github.com/feli05/autoscan/internal/tui/components"
	"github.com/feli05/autoscan/internal/tui/styles"
)

type State struct {
	Width        int
	ExportCursor int
	Report       *domain.RunReport
}

type UpdateResult struct {
	ExportCursor int
	GoBack       bool
	DoExport     bool
}

type DoneMsg struct {
	Format string
	Path   string
}

type ErrorMsg struct {
	Err error
}

func View(s State) string {
	var b strings.Builder

	b.WriteString(components.RenderHeader("Export Results"))

	boxWidth := s.Width - 8
	if boxWidth < 60 {
		boxWidth = 60
	}

	formats := []struct {
		name string
		ext  string
		desc string
	}{
		{
			name: "JSON",
			ext:  ".json",
			desc: "Structured data for scripts & tools",
		},
		{
			name: "CSV",
			ext:  ".csv",
			desc: "Import into Excel, Google Sheets",
		},
	}

	for i, f := range formats {
		borderColor := styles.Muted
		if i == s.ExportCursor {
			borderColor = styles.Primary
		}

		formatBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1, 2).
			Width(boxWidth)

		var content strings.Builder

		if i == s.ExportCursor {
			content.WriteString("▸ ")
			content.WriteString(styles.SelectedItem.Render(f.name))
		} else {
			content.WriteString("  ")
			content.WriteString(styles.NormalItem.Render(f.name))
		}
		content.WriteString(styles.SubtleText.Render(fmt.Sprintf("  (%s)", f.ext)))
		content.WriteString("\n")

		content.WriteString(styles.SubtleText.Render("  " + f.desc))

		b.WriteString(formatBox.Render(content.String()))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.SubtleText.Render("  Output: ./autoscan_report" + formats[s.ExportCursor].ext))
	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "export"},
		{Key: "esc", Desc: "back"},
	}))

	return b.String()
}

func Update(s State, msg tea.KeyMsg) UpdateResult {
	result := UpdateResult{
		ExportCursor: s.ExportCursor,
	}

	switch msg.String() {
	case "j", "down":
		if result.ExportCursor < 1 {
			result.ExportCursor++
		}
	case "k", "up":
		if result.ExportCursor > 0 {
			result.ExportCursor--
		}
	case "enter":
		if s.Report != nil {
			result.DoExport = true
		}
	case "q", "esc":
		result.GoBack = true
	}
	return result
}

func DoExport(report domain.RunReport, cursor int) tea.Cmd {
	return func() tea.Msg {
		outputDir := "."
		var path string
		var err error
		var format string

		switch cursor {
		case 0:
			format = "JSON"
			path, err = exportpkg.JSON(report, outputDir)
		case 1:
			format = "CSV"
			path, err = exportpkg.CSV(report, outputDir)
		}

		if err != nil {
			return ErrorMsg{Err: err}
		}
		return DoneMsg{Format: format, Path: path}
	}
}
