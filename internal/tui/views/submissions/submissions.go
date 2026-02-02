package submissions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/feli05/autoscan/internal/config"
	"github.com/feli05/autoscan/internal/domain"
	"github.com/feli05/autoscan/internal/tui/components"
	"github.com/feli05/autoscan/internal/tui/styles"
)

type SimilarityComputeState int

const (
	SimilarityNotStarted SimilarityComputeState = iota
	SimilarityComputing
	SimilarityDone
	SimilarityError
)

type SimilarityPair = domain.SimilarityPairResult

type PairDetailLoadedMsg struct {
	Process   string
	PairIndex int
	ContentA  []byte
	ContentB  []byte
	Err       error
	RunID     int64
}

type Navigation int

const (
	NavNone Navigation = iota
	NavGoHome
	NavGoDetails
	NavGoExport
	NavStartRun
)

type State struct {
	Width       int
	Height      int
	VisibleRows int

	PolicyName string
	Root       string

	Report    *domain.RunReport
	Results   []domain.SubmissionResult
	Filtered  []domain.SubmissionResult
	IsRunning bool
	RunError  string
	RunID     int64

	Settings config.Settings

	SubmissionsTab int
	Cursor         int
	ScrollOffset   int
	Filter         int

	SearchInput  textinput.Model
	SearchActive bool
	SearchQuery  string

	SimilarityProcessNames   []string
	SimilaritySelectedProc   int
	SimilarityPairsByProcess map[string][]SimilarityPair
	SimilarityStateByProcess map[string]SimilarityComputeState
	SimilarityErrorByProcess map[string]string
	SimilarityInFlight       map[string]bool
	SimilarityCursor         int
	SimilarityScroll         int

	PairDetailOpen        bool
	PairDetailProcess     string
	PairDetailPairIndex   int
	PairDetailContentA    []byte
	PairDetailContentB    []byte
	PairDetailLoadErr     string
	PairDetailFocusedPane int
	PairDetailScrollA     int
	PairDetailScrollB     int
	PairDetailHScrollA    int
	PairDetailHScrollB    int

	Spinner string

	SourceFileByProcess map[string]string
}

type UpdateResult struct {
	State

	Cmd                   tea.Cmd
	Nav                   Navigation
	ClearResults          bool
	ComputeSimilarityFor  string // Process name to compute similarity for (handlers.go will call the real function)
}

func Update(s State, msg tea.KeyMsg) UpdateResult {
	r := UpdateResult{State: s}

	if s.IsRunning {
		return r
	}

	if s.PairDetailOpen {
		return updatePairDetail(s, msg)
	}

	if msg.String() == "tab" {
		if s.Report != nil {
			r.SubmissionsTab = (s.SubmissionsTab + 1) % 2
			if r.SubmissionsTab == 0 {
				r.Cursor = 0
				r.ScrollOffset = 0
			} else {
				r.SimilarityCursor = 0
				r.SimilarityScroll = 0
				proc := currentSimilarityProcessName(r.State)
				if proc != "" && r.SimilarityStateByProcess[proc] == SimilarityNotStarted {
					r.ComputeSimilarityFor = proc
				}
			}
		}
		return r
	}

	if s.SubmissionsTab == 1 {
		return updateSimilarity(s, msg)
	}

	if s.SearchActive {
		return updateSearch(s, msg)
	}

	return updateResults(s, msg)
}

func updateSearch(s State, msg tea.KeyMsg) UpdateResult {
	r := UpdateResult{State: s}

	switch msg.String() {
	case "esc", "down", "j":
		r.SearchActive = false
		r.SearchQuery = s.SearchInput.Value()
		r.SearchInput = s.SearchInput
		r.SearchInput.Blur()
		r.Cursor = 0
		r.ScrollOffset = 0
		return r
	case "enter":
		r.SearchActive = false
		r.SearchQuery = s.SearchInput.Value()
		r.SearchInput = s.SearchInput
		r.SearchInput.Blur()
		return r
	}

	prev := s.SearchInput.Value()
	var cmd tea.Cmd
	newInput, cmd := s.SearchInput.Update(msg)
	r.SearchInput = newInput
	r.Cmd = cmd
	if newInput.Value() != prev {
		r.SearchQuery = newInput.Value()
		r.Cursor = 0
		r.ScrollOffset = 0
		r.ClearResults = true
	}
	return r
}

func updateResults(s State, msg tea.KeyMsg) UpdateResult {
	r := UpdateResult{State: s}
	filtered := s.Filtered

	switch msg.String() {
	case "/":
		r.SearchActive = true
		r.SearchInput = s.SearchInput
		r.SearchInput.Focus()
		r.Cmd = textinput.Blink
		return r
	case "esc":
		if strings.TrimSpace(s.SearchQuery) != "" {
			r.SearchQuery = ""
			r.SearchInput = s.SearchInput
			r.SearchInput.SetValue("")
			r.Cursor = 0
			r.ScrollOffset = 0
			r.ClearResults = true
			return r
		}
		r.Nav = NavGoHome
		return r
	case "j", "down":
		if s.Cursor < len(filtered)-1 {
			r.Cursor = s.Cursor + 1
			if r.Cursor >= s.ScrollOffset+s.VisibleRows {
				r.ScrollOffset = s.ScrollOffset + 1
			}
			r.ClearResults = true
		}
	case "k", "up":
		if s.Cursor > 0 {
			r.Cursor = s.Cursor - 1
			if r.Cursor < s.ScrollOffset {
				r.ScrollOffset = s.ScrollOffset - 1
			}
			r.ClearResults = true
		} else {
			r.SearchActive = true
			r.SearchInput = s.SearchInput
			r.SearchInput.Focus()
			r.Cmd = textinput.Blink
			return r
		}
	case "enter":
		if len(filtered) > 0 {
			r.Nav = NavGoDetails
		}
	case "f":
		r.Filter = (s.Filter + 1) % 4
		r.Cursor = 0
		r.ScrollOffset = 0
	case "r":
		r.Nav = NavStartRun
	case "e":
		r.Nav = NavGoExport
	case "q":
		r.Nav = NavGoHome
	}

	return r
}

func updateSimilarity(s State, msg tea.KeyMsg) UpdateResult {
	r := UpdateResult{State: s}

	switch msg.String() {
	case "esc", "q":
		r.Nav = NavGoHome
		r.SimilarityPairsByProcess = make(map[string][]SimilarityPair)
		r.SimilarityStateByProcess = make(map[string]SimilarityComputeState)
		r.SimilarityErrorByProcess = make(map[string]string)
		return r
	}

	if len(s.SimilarityProcessNames) == 0 {
		return r
	}

	currentProc := currentSimilarityProcessName(s)
	if currentProc != "" && s.SimilarityStateByProcess[currentProc] == SimilarityNotStarted {
		r.ComputeSimilarityFor = currentProc
		return r
	}

	pairs := s.SimilarityPairsByProcess[currentProc]
	dataRows := min(30, s.VisibleRows-1)
	if dataRows < 6 {
		dataRows = 6
	}

	switch msg.String() {
	case "j", "down":
		if len(pairs) == 0 {
			return r
		}
		if s.SimilarityCursor < len(pairs)-1 {
			r.SimilarityCursor = s.SimilarityCursor + 1
			if r.SimilarityCursor >= s.SimilarityScroll+dataRows {
				r.SimilarityScroll = s.SimilarityScroll + 1
			}
		}
	case "k", "up":
		if len(pairs) == 0 {
			return r
		}
		if s.SimilarityCursor > 0 {
			r.SimilarityCursor = s.SimilarityCursor - 1
			if r.SimilarityCursor < s.SimilarityScroll {
				r.SimilarityScroll = s.SimilarityScroll - 1
			}
		}
	case "l", "right":
		if s.SimilaritySelectedProc < len(s.SimilarityProcessNames)-1 {
			r.SimilaritySelectedProc = s.SimilaritySelectedProc + 1
			r.SimilarityCursor = 0
			r.SimilarityScroll = 0
			proc := currentSimilarityProcessName(r.State)
			if proc != "" && r.SimilarityStateByProcess[proc] == SimilarityNotStarted {
				r.ComputeSimilarityFor = proc
			}
		}
	case "h", "left":
		if s.SimilaritySelectedProc > 0 {
			r.SimilaritySelectedProc = s.SimilaritySelectedProc - 1
			r.SimilarityCursor = 0
			r.SimilarityScroll = 0
			proc := currentSimilarityProcessName(r.State)
			if proc != "" && r.SimilarityStateByProcess[proc] == SimilarityNotStarted {
				r.ComputeSimilarityFor = proc
			}
		}
	case "enter":
		if len(pairs) == 0 || s.SimilarityCursor >= len(pairs) {
			return r
		}
		pair := pairs[s.SimilarityCursor]
		srcFile := ""
		if s.SourceFileByProcess != nil {
			srcFile = s.SourceFileByProcess[currentProc]
		}
		if srcFile == "" && len(s.Results) > 0 && len(s.Results[0].Submission.CFiles) > 0 {
			srcFile = s.Results[0].Submission.CFiles[0]
		}
		if srcFile == "" {
			return r
		}
		pathA := filepath.Join(s.Results[pair.AIndex].Submission.Path, srcFile)
		pathB := filepath.Join(s.Results[pair.BIndex].Submission.Path, srcFile)
		r.PairDetailOpen = true
		r.PairDetailProcess = currentProc
		r.PairDetailPairIndex = s.SimilarityCursor
		r.PairDetailContentA = nil
		r.PairDetailContentB = nil
		r.PairDetailLoadErr = ""
		r.PairDetailFocusedPane = 0
		r.PairDetailScrollA = 0
		r.PairDetailScrollB = 0
		r.PairDetailHScrollA = 0
		r.PairDetailHScrollB = 0
		r.Cmd = loadPairDetailFiles(currentProc, s.SimilarityCursor, pathA, pathB, s.RunID)
	}
	return r
}

func updatePairDetail(s State, msg tea.KeyMsg) UpdateResult {
	r := UpdateResult{State: s}

	pairs := s.SimilarityPairsByProcess[s.PairDetailProcess]
	if s.PairDetailPairIndex >= len(pairs) {
		r.PairDetailOpen = false
		return r
	}

	const pairDetailMaxPaneHeight = 30
	paneHeight := pairDetailMaxPaneHeight
	if s.VisibleRows < paneHeight {
		paneHeight = s.VisibleRows
	}
	if paneHeight < 8 {
		paneHeight = 8
	}

	linesA := len(strings.Split(string(s.PairDetailContentA), "\n"))
	linesB := len(strings.Split(string(s.PairDetailContentB), "\n"))
	maxScrollA := max(0, linesA-paneHeight)
	maxScrollB := max(0, linesB-paneHeight)
	_, contentWidth := pairDetailPaneWidths(s.Width)
	maxHScrollA := max(0, maxDisplayWidthForContent(s.PairDetailContentA)-contentWidth)
	maxHScrollB := max(0, maxDisplayWidthForContent(s.PairDetailContentB)-contentWidth)

	switch msg.String() {
	case "esc", "q":
		r.PairDetailOpen = false
		return r
	case "h":
		r.PairDetailFocusedPane = 0
		return r
	case "l":
		r.PairDetailFocusedPane = 1
		return r
	case "down":
		if s.PairDetailFocusedPane == 0 {
			if s.PairDetailScrollA < maxScrollA {
				r.PairDetailScrollA = s.PairDetailScrollA + 1
			}
		} else {
			if s.PairDetailScrollB < maxScrollB {
				r.PairDetailScrollB = s.PairDetailScrollB + 1
			}
		}
	case "up":
		if s.PairDetailFocusedPane == 0 {
			if s.PairDetailScrollA > 0 {
				r.PairDetailScrollA = s.PairDetailScrollA - 1
			}
		} else {
			if s.PairDetailScrollB > 0 {
				r.PairDetailScrollB = s.PairDetailScrollB - 1
			}
		}
	case "right":
		if s.PairDetailFocusedPane == 0 {
			if s.PairDetailHScrollA < maxHScrollA {
				r.PairDetailHScrollA = s.PairDetailHScrollA + 1
			}
		} else {
			if s.PairDetailHScrollB < maxHScrollB {
				r.PairDetailHScrollB = s.PairDetailHScrollB + 1
			}
		}
	case "left":
		if s.PairDetailFocusedPane == 0 {
			if s.PairDetailHScrollA > 0 {
				r.PairDetailHScrollA = s.PairDetailHScrollA - 1
			}
		} else {
			if s.PairDetailHScrollB > 0 {
				r.PairDetailHScrollB = s.PairDetailHScrollB - 1
			}
		}
	}
	return r
}

func currentSimilarityProcessName(s State) string {
	if len(s.SimilarityProcessNames) == 0 {
		return ""
	}
	if s.SimilaritySelectedProc < 0 || s.SimilaritySelectedProc >= len(s.SimilarityProcessNames) {
		return s.SimilarityProcessNames[0]
	}
	return s.SimilarityProcessNames[s.SimilaritySelectedProc]
}

func loadPairDetailFiles(process string, pairIndex int, pathA, pathB string, runID int64) tea.Cmd {
	return func() tea.Msg {
		contentA, errA := os.ReadFile(pathA)
		if errA != nil {
			return PairDetailLoadedMsg{Process: process, PairIndex: pairIndex, ContentA: nil, ContentB: nil, Err: errA, RunID: runID}
		}
		contentB, errB := os.ReadFile(pathB)
		if errB != nil {
			return PairDetailLoadedMsg{Process: process, PairIndex: pairIndex, ContentA: contentA, ContentB: nil, Err: errB, RunID: runID}
		}
		return PairDetailLoadedMsg{Process: process, PairIndex: pairIndex, ContentA: contentA, ContentB: contentB, Err: nil, RunID: runID}
	}
}

func maxDisplayWidthForContent(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	maxWidth := 0
	for _, line := range strings.Split(string(content), "\n") {
		line = components.SanitizeDisplay(line)
		line = expandTabsForPane(line, pairDetailTabWidth)
		if w := lipgloss.Width(line); w > maxWidth {
			maxWidth = w
		}
	}
	return maxWidth
}

func View(s State) string {
	var b strings.Builder

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		Padding(0, 2)

	b.WriteString("\n")
	b.WriteString(header.Render(s.PolicyName))
	b.WriteString("\n")
	b.WriteString(styles.SubtleText.Render(fmt.Sprintf("  %s", s.Root)))
	b.WriteString("\n\n")

	tabs := []string{"Results", "Similarity"}
	var tabRow strings.Builder
	tabRow.WriteString("  ")
	for i, tab := range tabs {
		if i == s.SubmissionsTab {
			tabRow.WriteString(styles.TabActive.Render(fmt.Sprintf(" %s ", tab)))
		} else {
			tabRow.WriteString(styles.TabInactive.Render(fmt.Sprintf(" %s ", tab)))
		}
		tabRow.WriteString(" ")
	}
	b.WriteString(tabRow.String())
	b.WriteString("\n")

	if s.RunError != "" {
		b.WriteString("\n")
		errorBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Error).
			Padding(1, 2)
		b.WriteString(errorBox.Render(styles.ErrorText.Render("Error: " + s.RunError)))
		b.WriteString("\n")
	} else if s.IsRunning {
		b.WriteString(fmt.Sprintf("\n  %s Scanning and compiling...\n", s.Spinner))
	} else if s.Report != nil {
		b.WriteString(renderHeaderBox(s))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	if s.PairDetailOpen {
		b.WriteString(renderPairDetail(s))
		return b.String()
	}
	if s.SubmissionsTab == 0 {
		b.WriteString(renderResults(s))
	} else {
		b.WriteString(renderSimilarity(s))
	}

	return b.String()
}

func renderHeaderBox(s State) string {
	statsBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(0, 2).
		MarginTop(1)

	if s.SubmissionsTab == 0 {
		searchLabel := ""
		if strings.TrimSpace(s.SearchQuery) != "" {
			searchLabel = fmt.Sprintf("  Search: %s", s.SearchQuery)
		}

		filterStr := filterString(s.Filter)
		stats := fmt.Sprintf(
			"Pass: %d  Fail: %d  Banned: %d  Time: %dms  Filter: %s%s",
			s.Report.Summary.CompilePass,
			s.Report.Summary.CompileFail,
			s.Report.Summary.SubmissionsWithBanned,
			s.Report.Summary.DurationMs,
			filterStr,
			searchLabel,
		)
		return statsBox.Render(stats)
	}

	if len(s.SimilarityProcessNames) == 0 {
		return statsBox.Render("Similarity: no processes configured")
	}

	line2 := fmt.Sprintf(
		"Window size: %d   Min tokens: %d   Threshold: %.2f",
		s.Settings.PlagiarismWindowSize,
		s.Settings.PlagiarismMinFuncTokens,
		s.Settings.PlagiarismScoreThreshold,
	)

	return statsBox.Render(styles.NormalItem.Render(line2))
}

func filterString(f int) string {
	switch f {
	case 1:
		return "Compile Fails"
	case 2:
		return "Banned Calls"
	case 3:
		return "Clean"
	default:
		return "All"
	}
}

func renderResults(s State) string {
	var b strings.Builder

	searchBorderColor := styles.Muted
	if s.SearchActive {
		searchBorderColor = styles.Primary
	}
	searchBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(searchBorderColor).
		Padding(0, 2)
	b.WriteString(searchBox.Render(fmt.Sprintf("Search: %s", s.SearchInput.View())))
	b.WriteString("\n\n")

	tableBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(0, 1)

	var table strings.Builder

	const (
		colStatus  = 5
		colCompile = 10
		colBanned  = 10
		colGrade   = 8
	)
	fixedCols := colStatus + colCompile + colBanned + colGrade + 15
	colSubmission := s.Width - fixedCols
	if colSubmission < 30 {
		colSubmission = 30
	}
	if colSubmission > 80 {
		colSubmission = 80
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary)

	table.WriteString(headerStyle.Render(fmt.Sprintf("  %-*s %-*s  %-*s  %-*s  %-*s",
		colStatus, "",
		colSubmission, "Submission",
		colCompile, "Compile",
		colBanned, "Banned",
		colGrade, "Grade")))
	table.WriteString("\n")
	table.WriteString(strings.Repeat("─", 2+colStatus+1+colSubmission+2+colCompile+2+colBanned+2+colGrade))
	table.WriteString("\n")

	filtered := s.Filtered

	if len(filtered) == 0 && !s.IsRunning {
		table.WriteString(styles.SubtleText.Render("  No submissions found"))
		table.WriteString("\n")
	}

	endIdx := s.ScrollOffset + s.VisibleRows
	if endIdx > len(filtered) {
		endIdx = len(filtered)
	}

	for i := s.ScrollOffset; i < endIdx; i++ {
		r := filtered[i]

		var cursor string
		if i == s.Cursor {
			cursor = styles.Highlight.Render("▶ ")
		} else {
			cursor = "  "
		}

		var statusText string
		var statusStyled string
		switch r.Status {
		case domain.StatusClean:
			statusText = "[OK]"
			statusStyled = styles.SuccessText.Render(statusText)
		case domain.StatusBanned:
			statusText = "[!]"
			statusStyled = styles.WarningText.Render(statusText)
		case domain.StatusFailed, domain.StatusTimedOut:
			statusText = "[X]"
			statusStyled = styles.ErrorText.Render(statusText)
		default:
			statusText = "..."
			statusStyled = statusText
		}
		statusPadding := strings.Repeat(" ", max(0, colStatus-lipgloss.Width(statusText)))

		id := r.Submission.ID
		if s.Settings.ShortNames {
			if idx := strings.Index(id, "_"); idx > 0 {
				id = id[:idx]
			}
		}
		if lipgloss.Width(id) > colSubmission {
			runes := []rune(id)
			for lipgloss.Width(string(runes)) > colSubmission-3 && len(runes) > 0 {
				runes = runes[:len(runes)-1]
			}
			id = string(runes) + "..."
		}
		idPadding := strings.Repeat(" ", max(0, colSubmission-lipgloss.Width(id)))

		var compileText, compileStyled string
		if r.Compile.TimedOut {
			compileText = "TIMEOUT"
			compileStyled = styles.WarningText.Render(compileText)
		} else if !r.Compile.OK {
			compileText = "FAIL"
			compileStyled = styles.ErrorText.Render(compileText)
		} else {
			compileText = "OK"
			compileStyled = styles.SuccessText.Render(compileText)
		}
		compilePadding := strings.Repeat(" ", colCompile-len(compileText))

		var bannedText, bannedStyled string
		if r.Scan.TotalHits() > 0 {
			bannedText = fmt.Sprintf("%d", r.Scan.TotalHits())
			bannedStyled = styles.WarningText.Render(bannedText)
		} else {
			bannedText = "-"
			bannedStyled = bannedText
		}
		bannedPadding := strings.Repeat(" ", colBanned-len(bannedText))

		var gradeText, gradeStyled string
		if !r.Compile.OK || r.Compile.TimedOut || r.Scan.TotalHits() > 0 {
			gradeText = "2"
			gradeStyled = styles.ErrorText.Render(gradeText)
		} else {
			gradeText = "CHECK"
			gradeStyled = styles.SuccessText.Render(gradeText)
		}

		table.WriteString(fmt.Sprintf("%s%s%s %s%s  %s%s  %s%s  %s\n",
			cursor,
			statusStyled, statusPadding,
			id, idPadding,
			compileStyled, compilePadding,
			bannedStyled, bannedPadding,
			gradeStyled))
	}

	if len(filtered) > s.VisibleRows {
		table.WriteString(styles.SubtleText.Render(fmt.Sprintf("\n  Showing %d-%d of %d",
			s.ScrollOffset+1, endIdx, len(filtered))))
	}

	b.WriteString(tableBox.Render(table.String()))

	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{Key: "tab", Desc: "switch to similarity"},
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "details"},
		{Key: "/", Desc: "search"},
		{Key: "f", Desc: "filter"},
		{Key: "r", Desc: "re-run"},
		{Key: "e", Desc: "export"},
		{Key: "esc", Desc: "clear/back"},
	}))

	return b.String()
}

func renderSimilarity(s State) string {
	var b strings.Builder

	if s.Report == nil {
		b.WriteString(styles.SubtleText.Render("No run data available. Run the grader first."))
		return b.String()
	}

	if len(s.SimilarityProcessNames) == 0 {
		b.WriteString(styles.SubtleText.Render("No processes configured. Check policy configuration."))
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString("  ")
	b.WriteString(styles.SubtleText.Render("Process: "))
	for i, name := range s.SimilarityProcessNames {
		if i == s.SimilaritySelectedProc {
			b.WriteString(styles.TabActive.Render(fmt.Sprintf(" %s ", name)))
		} else {
			b.WriteString(styles.TabInactive.Render(fmt.Sprintf(" %s ", name)))
		}
		b.WriteString(" ")
	}
	b.WriteString("\n\n")

	currentProc := currentSimilarityProcessName(s)
	pairs := s.SimilarityPairsByProcess[currentProc]
	state := s.SimilarityStateByProcess[currentProc]

	if errText, ok := s.SimilarityErrorByProcess[currentProc]; ok && errText != "" {
		b.WriteString(styles.WarningText.Render("Similarity error: " + errText))
		b.WriteString("\n")
		return b.String()
	}

	if state == SimilarityNotStarted || state == SimilarityComputing {
		b.WriteString(styles.SubtleText.Render("Computing similarity..."))
		b.WriteString("\n")
		return b.String()
	}
	if len(pairs) == 0 {
		b.WriteString(styles.SubtleText.Render("No pairs found (not enough comparable submissions)."))
		b.WriteString("\n")
		return b.String()
	}

	tableBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(0, 1)

	var table strings.Builder

	padOrTrim := func(str string, w int) string {
		str = components.TruncateToWidth(str, w)
		if d := w - lipgloss.Width(str); d > 0 {
			str += strings.Repeat(" ", d)
		}
		return str
	}

	const (
		colRank    = 5
		colJac     = 9
		colPerFunc = 9
		colMatches = 13
		colStatus  = 8
	)
	fixedCols := 2 + colRank + colJac + colPerFunc + colMatches + colStatus + 7
	availForNames := s.Width - fixedCols
	colSub := 20
	colSubB := 20
	if availForNames > 0 {
		per := availForNames / 2
		if per > colSub {
			colSub = per
			colSubB = per
		}
	}
	if colSub > 34 {
		colSub = 34
	}
	if colSubB > 34 {
		colSubB = 34
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary)

	headerLine := "  " +
		padOrTrim("#", colRank) + " " +
		padOrTrim("Submission A", colSub) + " " +
		padOrTrim("Submission B", colSubB) + " " +
		padOrTrim("Jaccard", colJac) + " " +
		padOrTrim("Per-func", colPerFunc) + " " +
		padOrTrim("Matches", colMatches) + " " +
		padOrTrim("Status", colStatus)
	table.WriteString(headerStyle.Render(headerLine))
	table.WriteString("\n")
	table.WriteString(strings.Repeat("─", 2+colRank+1+colSub+1+colSubB+1+colJac+1+colPerFunc+1+colMatches+1+colStatus))
	table.WriteString("\n")

	dataRows := min(30, s.VisibleRows-1)
	if dataRows < 6 {
		dataRows = 6
	}

	endIdx := s.SimilarityScroll + dataRows
	if endIdx > len(pairs) {
		endIdx = len(pairs)
	}

	for i := s.SimilarityScroll; i < endIdx; i++ {
		p := pairs[i]
		res := p.Result

		cursor := "  "
		if i == s.SimilarityCursor {
			cursor = styles.Highlight.Render("▶ ")
		}

		rank := i + 1
		aID := s.Results[p.AIndex].Submission.ID
		bID := s.Results[p.BIndex].Submission.ID
		if s.Settings.ShortNames {
			if idx := strings.Index(aID, "_"); idx > 0 {
				aID = aID[:idx]
			}
			if idx := strings.Index(bID, "_"); idx > 0 {
				bID = bID[:idx]
			}
		}
		if lipgloss.Width(aID) > colSub {
			runes := []rune(aID)
			for lipgloss.Width(string(runes)) > colSub-3 && len(runes) > 0 {
				runes = runes[:len(runes)-1]
			}
			aID = string(runes) + "..."
		}
		if lipgloss.Width(bID) > colSubB {
			runes := []rune(bID)
			for lipgloss.Width(string(runes)) > colSubB-3 && len(runes) > 0 {
				runes = runes[:len(runes)-1]
			}
			bID = string(runes) + "..."
		}

		statusText := "OK"
		if res.Flagged {
			statusText = "FLAG"
		}

		matchesText := fmt.Sprintf("%d/%d", res.WindowMatches, res.WindowUnion)
		jacText := fmt.Sprintf("%.2f%%", res.WindowJaccard*100)
		perFuncText := fmt.Sprintf("%.2f%%", res.PerFuncSimilarity*100)

		statusPadded := padOrTrim(statusText, colStatus)
		statusRendered := styles.SuccessText.Render(statusPadded)
		if res.Flagged {
			statusRendered = styles.WarningText.Render(statusPadded)
		}

		row := cursor +
			padOrTrim(fmt.Sprintf("%d", rank), colRank) + " " +
			padOrTrim(aID, colSub) + " " +
			padOrTrim(bID, colSubB) + " " +
			padOrTrim(jacText, colJac) + " " +
			padOrTrim(perFuncText, colPerFunc) + " " +
			padOrTrim(matchesText, colMatches) + " " +
			statusRendered
		table.WriteString(row)
		table.WriteString("\n")
	}

	for i := endIdx; i < s.SimilarityScroll+dataRows; i++ {
		table.WriteString("\n")
	}

	footer := ""
	if len(pairs) > dataRows {
		footer = fmt.Sprintf("  Showing %d-%d of %d", s.SimilarityScroll+1, endIdx, len(pairs))
	}
	table.WriteString(styles.SubtleText.Render(padOrTrim(footer, 2+colRank+1+colSub+1+colSubB+1+colJac+1+colPerFunc+1+colMatches+1+colStatus)))

	b.WriteString(tableBox.Render(table.String()))
	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{Key: "tab", Desc: "switch to results"},
		{Key: "h/l", Desc: "prev/next process"},
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "pair detail"},
		{Key: "esc", Desc: "back"},
	}))

	return b.String()
}

var pairDetailHighlightStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#334155")).
	Foreground(styles.Text)

func renderPairDetail(s State) string {
	var b strings.Builder

	pairs := s.SimilarityPairsByProcess[s.PairDetailProcess]
	if s.PairDetailPairIndex >= len(pairs) {
		b.WriteString(styles.SubtleText.Render("No pair selected."))
		return b.String()
	}
	pair := pairs[s.PairDetailPairIndex]
	res := pair.Result

	if s.PairDetailLoadErr != "" {
		b.WriteString(styles.WarningText.Render("Error: " + s.PairDetailLoadErr))
		b.WriteString("\n\n")
		b.WriteString(components.RenderHelpBar([]components.HelpItem{{Key: "esc", Desc: "back"}}))
		return b.String()
	}
	if s.PairDetailContentA == nil {
		b.WriteString(styles.SubtleText.Render("Loading files..."))
		b.WriteString("\n\n")
		b.WriteString(components.RenderHelpBar([]components.HelpItem{{Key: "esc", Desc: "back"}}))
		return b.String()
	}

	statsBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(0, 2).
		MarginTop(0)
	nameA := s.Results[pair.AIndex].Submission.ID
	nameB := s.Results[pair.BIndex].Submission.ID
	if s.Settings.ShortNames {
		if idx := strings.Index(nameA, "_"); idx > 0 {
			nameA = nameA[:idx]
		}
		if idx := strings.Index(nameB, "_"); idx > 0 {
			nameB = nameB[:idx]
		}
	}
	summary := fmt.Sprintf(
		"%s  vs  %s   ·   Jaccard: %.2f%%   Per-func: %.2f%%   Matches: %d/%d",
		nameA, nameB,
		res.WindowJaccard*100, res.PerFuncSimilarity*100,
		res.WindowMatches, res.WindowUnion,
	)
	b.WriteString(statsBox.Render(summary))
	b.WriteString("\n\n")

	const pairDetailMaxPaneHeight = 30
	paneHeight := min(s.VisibleRows, pairDetailMaxPaneHeight)
	if paneHeight < 8 {
		paneHeight = 8
	}

	var spansA, spansB []domain.MatchSpan
	for _, wm := range res.Matches {
		spansA = append(spansA, wm.SpansA...)
		spansB = append(spansB, wm.SpansB...)
	}

	halfWidth, contentWidth := pairDetailPaneWidths(s.Width)

	leftPane := renderCodePane(s.PairDetailContentA, spansA, s.PairDetailScrollA, s.PairDetailHScrollA, paneHeight, contentWidth)
	rightPane := renderCodePane(s.PairDetailContentB, spansB, s.PairDetailScrollB, s.PairDetailHScrollB, paneHeight, contentWidth)

	lineStyle := lipgloss.NewStyle().Width(contentWidth).MaxWidth(contentWidth)
	leftLines := strings.Split(strings.TrimSuffix(leftPane, "\n"), "\n")
	rightLines := strings.Split(strings.TrimSuffix(rightPane, "\n"), "\n")
	for len(leftLines) < paneHeight {
		leftLines = append(leftLines, strings.Repeat(" ", contentWidth))
	}
	for len(rightLines) < paneHeight {
		rightLines = append(rightLines, strings.Repeat(" ", contentWidth))
	}

	var leftContent, rightContent strings.Builder
	for i := 0; i < paneHeight; i++ {
		leftContent.WriteString(lineStyle.Render(leftLines[i]))
		leftContent.WriteString("\n")
		rightContent.WriteString(lineStyle.Render(rightLines[i]))
		rightContent.WriteString("\n")
	}

	leftBorderColor := styles.Muted
	rightBorderColor := styles.Muted
	if s.PairDetailFocusedPane == 0 {
		leftBorderColor = styles.Primary
	} else {
		rightBorderColor = styles.Primary
	}

	leftBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(leftBorderColor).
		Padding(0, 1).
		Width(halfWidth).
		Render(strings.TrimSuffix(leftContent.String(), "\n"))
	rightBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(rightBorderColor).
		Padding(0, 1).
		Width(halfWidth).
		Render(strings.TrimSuffix(rightContent.String(), "\n"))

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftBox, "  ", rightBox))

	matchInfo := ""
	if len(res.Matches) > 0 {
		matchInfo = fmt.Sprintf("  %d matches", len(res.Matches))
	}
	b.WriteString("\n\n")
	b.WriteString(components.RenderHelpBar([]components.HelpItem{
		{Key: "↑/↓", Desc: "scroll"},
		{Key: "←/→", Desc: "pan"},
		{Key: "h/l", Desc: "switch pane"},
		{Key: "esc", Desc: "back"},
	}))
	if matchInfo != "" {
		b.WriteString(styles.SubtleText.Render(matchInfo))
	}
	return b.String()
}

const pairDetailTabWidth = 8

func pairDetailPaneWidths(totalWidth int) (halfWidth, contentWidth int) {
	halfWidth = (totalWidth - 6) / 2
	if halfWidth < 20 {
		halfWidth = 20
	}
	contentWidth = halfWidth - 4
	if contentWidth < 10 {
		contentWidth = 10
	}
	return halfWidth, contentWidth
}

func expandTabsForPane(str string, width int) string {
	if !strings.Contains(str, "\t") {
		return str
	}
	var b strings.Builder
	b.Grow(len(str))
	col := 0
	for _, r := range str {
		if r == '\t' {
			spaces := width - (col % width)
			if spaces == 0 {
				spaces = width
			}
			b.WriteString(strings.Repeat(" ", spaces))
			col += spaces
			continue
		}
		b.WriteRune(r)
		col += lipgloss.Width(string(r))
	}
	return b.String()
}

func sliceDisplayWindow(str string, start, width int) string {
	if width <= 0 || str == "" {
		return ""
	}
	if start < 0 {
		start = 0
	}
	var b strings.Builder
	b.Grow(len(str))
	col := 0
	for _, r := range str {
		rw := lipgloss.Width(string(r))
		if col+rw <= start {
			col += rw
			continue
		}
		if col-start+rw > width {
			break
		}
		b.WriteRune(r)
		col += rw
	}
	return b.String()
}

func byteColToDisplayCol(line string, byteCol, tabWidth int) int {
	if byteCol <= 1 {
		return 1
	}
	target := byteCol - 1
	col := 1
	for idx, r := range line {
		if idx >= target {
			break
		}
		if r == '\t' {
			spaces := tabWidth - ((col - 1) % tabWidth)
			if spaces == 0 {
				spaces = tabWidth
			}
			col += spaces
			continue
		}
		col += lipgloss.Width(string(r))
	}
	return col
}

func renderCodePane(content []byte, spans []domain.MatchSpan, scrollLine, hScroll, height, width int) string {
	lines := strings.Split(string(content), "\n")
	blankLine := strings.Repeat(" ", width) + "\n"

	var b strings.Builder
	contentLines := 0
	if scrollLine < len(lines) {
		end := scrollLine + height
		if end > len(lines) {
			end = len(lines)
		}
		contentLines = end - scrollLine
		for i := scrollLine; i < end; i++ {
			rawLine := components.SanitizeDisplay(lines[i])
			line := expandTabsForPane(rawLine, pairDetailTabWidth)
			fullLineWidth := lipgloss.Width(line)
			line = sliceDisplayWindow(line, hScroll, width)
			lineNum1 := i + 1
			runes := []rune(line)
			lineRuneLen := len(runes)
			var ranges [][2]int
			for _, sp := range spans {
				if sp.EndLine < lineNum1 || sp.StartLine > lineNum1 {
					continue
				}
				startCol := byteColToDisplayCol(rawLine, sp.StartCol, pairDetailTabWidth)
				endCol := byteColToDisplayCol(rawLine, sp.EndCol, pairDetailTabWidth)
				if sp.StartLine == lineNum1 && sp.EndLine == lineNum1 {
					// segment on this line
				} else if sp.StartLine == lineNum1 {
					endCol = fullLineWidth + 1
				} else if sp.EndLine == lineNum1 {
					startCol = 1
				} else {
					startCol = 1
					endCol = fullLineWidth + 1
				}
				startIdx := (startCol - 1) - hScroll
				if startIdx < 0 {
					startIdx = 0
				}
				endIdx := endCol - hScroll
				if endIdx > lineRuneLen {
					endIdx = lineRuneLen
				}
				if startIdx < endIdx {
					ranges = append(ranges, [2]int{startIdx, endIdx})
				}
			}
			merged := mergeRanges(ranges)
			if len(merged) == 0 {
				b.WriteString(string(runes))
			} else {
				last := 0
				for _, r := range merged {
					if r[0] > last {
						b.WriteString(string(runes[last:r[0]]))
					}
					seg := string(runes[r[0]:r[1]])
					b.WriteString(pairDetailHighlightStyle.Render(seg))
					last = r[1]
				}
				if last < lineRuneLen {
					b.WriteString(string(runes[last:]))
				}
			}
			b.WriteString("\n")
		}
	}

	for i := contentLines; i < height; i++ {
		b.WriteString(blankLine)
	}
	return b.String()
}

func mergeRanges(ranges [][2]int) [][2]int {
	if len(ranges) == 0 {
		return nil
	}
	type span struct{ start, end int }
	sorted := make([]span, len(ranges))
	for i, r := range ranges {
		sorted[i] = span{r[0], r[1]}
	}
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].start < sorted[i].start {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	var result [][2]int
	cur := sorted[0]
	for i := 1; i < len(sorted); i++ {
		if sorted[i].start <= cur.end {
			if sorted[i].end > cur.end {
				cur.end = sorted[i].end
			}
		} else {
			result = append(result, [2]int{cur.start, cur.end})
			cur = sorted[i]
		}
	}
	result = append(result, [2]int{cur.start, cur.end})
	return result
}
