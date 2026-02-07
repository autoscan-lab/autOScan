package details

import (
	"fmt"
	"strings"

	"github.com/felitrejos/autoscan/internal/tui/components"
)

func renderFilesTab(s State) string {
	r := s.Result
	var b strings.Builder

	b.WriteString(components.SubtleText.Render(fmt.Sprintf("%d source file(s)", len(r.Submission.CFiles))))
	b.WriteString("\n\n")

	for _, f := range r.Submission.CFiles {
		b.WriteString(fmt.Sprintf("  %s\n", f))
	}

	if len(r.Scan.ParseErrors) > 0 {
		b.WriteString("\n")
		b.WriteString(components.WarningText.Render("Parse errors:"))
		b.WriteString("\n")
		for _, e := range r.Scan.ParseErrors {
			b.WriteString(fmt.Sprintf("  - %s\n", e))
		}
	}

	return b.String()
}
