package details

import (
	"fmt"
	"strings"

	"github.com/feli05/autoscan/internal/tui/styles"
)

func renderFilesTab(s State) string {
	r := s.Result
	var b strings.Builder

	b.WriteString(styles.SubtleText.Render(fmt.Sprintf("%d source file(s)", len(r.Submission.CFiles))))
	b.WriteString("\n\n")

	for _, f := range r.Submission.CFiles {
		b.WriteString(fmt.Sprintf("  %s\n", f))
	}

	if len(r.Scan.ParseErrors) > 0 {
		b.WriteString("\n")
		b.WriteString(styles.WarningText.Render("Parse errors:"))
		b.WriteString("\n")
		for _, e := range r.Scan.ParseErrors {
			b.WriteString(fmt.Sprintf("  - %s\n", e))
		}
	}

	return b.String()
}
