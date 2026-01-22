package domain

// BannedHit represents a single occurrence of a banned function call.
type BannedHit struct {
	// Function is the name of the banned function (e.g., "printf")
	Function string

	// File is the source file where the call was found
	File string

	// Line is the 1-based line number
	Line int

	// Column is the 1-based column number
	Column int

	// Snippet is a short code excerpt around the call
	Snippet string
}

// NewBannedHit creates a new BannedHit.
func NewBannedHit(function, file string, line, column int, snippet string) BannedHit {
	return BannedHit{
		Function: function,
		File:     file,
		Line:     line,
		Column:   column,
		Snippet:  snippet,
	}
}

// ScanResult holds all banned function call hits for a submission.
type ScanResult struct {
	// Hits is the list of individual banned call occurrences
	Hits []BannedHit

	// HitsByFunction groups hits by function name for summary display
	HitsByFunction map[string][]BannedHit

	// ParseErrors lists files that couldn't be parsed
	ParseErrors []string
}

// NewScanResult creates a new ScanResult from a list of hits.
func NewScanResult(hits []BannedHit, parseErrors []string) ScanResult {
	byFunc := make(map[string][]BannedHit)
	for _, h := range hits {
		byFunc[h.Function] = append(byFunc[h.Function], h)
	}
	return ScanResult{
		Hits:           hits,
		HitsByFunction: byFunc,
		ParseErrors:    parseErrors,
	}
}

// TotalHits returns the total number of banned call hits.
func (s ScanResult) TotalHits() int {
	return len(s.Hits)
}

// UniqueFunctions returns the number of unique banned functions used.
func (s ScanResult) UniqueFunctions() int {
	return len(s.HitsByFunction)
}
