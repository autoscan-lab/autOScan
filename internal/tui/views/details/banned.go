package details

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/autoscan-lab/autoscan/internal/tui/components"
)

func renderBannedTab(s State) string {
	r := s.Result
	var b strings.Builder

	if r.Scan.TotalHits() == 0 {
		b.WriteString(components.SuccessText.Render("[OK] No banned function calls detected"))
		return b.String()
	}

	b.WriteString(components.WarningText.Render(fmt.Sprintf("[!] %d banned call(s) found", r.Scan.TotalHits())))
	b.WriteString("\n\n")

	var funcNames []string
	for fn := range r.Scan.HitsByFunction {
		funcNames = append(funcNames, fn)
	}
	sort.Strings(funcNames)

	for i, fn := range funcNames {
		hits := r.Scan.HitsByFunction[fn]
		expanded := s.ExpandedFuncs != nil && s.ExpandedFuncs[fn]

		arrow := "[+]"
		if expanded {
			arrow = "[-]"
		}

		var line string
		if i == s.BannedCursor {
			line = "> " + components.Highlight.Render(fmt.Sprintf("%s %s (%d)", arrow, fn, len(hits)))
		} else {
			line = fmt.Sprintf("  %s %s (%d)", arrow, fn, len(hits))
		}
		b.WriteString(line)
		b.WriteString("\n")

		if expanded {
			showMax := 5
			maxLineWidth := 65
			for j, hit := range hits {
				if j >= showMax {
					remaining := len(hits) - showMax
					b.WriteString(components.SubtleText.Render(fmt.Sprintf("       ... and %d more calls", remaining)))
					b.WriteString("\n")
					break
				}
				hitLine := fmt.Sprintf("       %s:%d %s", hit.File, hit.Line, hit.Snippet)
				if lipgloss.Width(hitLine) > maxLineWidth {
					runes := []rune(hitLine)
					for lipgloss.Width(string(runes)) > maxLineWidth-3 && len(runes) > 0 {
						runes = runes[:len(runes)-1]
					}
					hitLine = string(runes) + "..."
				}
				b.WriteString(components.SubtleText.Render(hitLine))
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

func getBannedFuncNames(s State) []string {
	var funcNames []string
	for fn := range s.Result.Scan.HitsByFunction {
		funcNames = append(funcNames, fn)
	}
	sort.Strings(funcNames)
	return funcNames
}
