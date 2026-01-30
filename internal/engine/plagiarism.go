package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/feli05/autoscan/internal/domain"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/c"
)

// FingerprintFile parses a C file and creates a fingerprint.
func FingerprintFile(path string, cfg domain.CompareConfig) (domain.FileFingerprint, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return domain.FileFingerprint{}, err
	}

	parser := sitter.NewParser()
	parser.SetLanguage(c.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return domain.FileFingerprint{}, err
	}
	defer tree.Close()

	fp := domain.FileFingerprint{
		FunctionHashes:  make(map[string]struct{}),
		WindowHashes:    make(map[string]struct{}),
		WindowSpans:     make(map[string][]domain.Span),
		FunctionWindows: make([]map[string]struct{}, 0),
		Content:         content,
		LineOffsets:     buildLineOffsets(content),
	}

	var funcs []*sitter.Node
	collectFunctionDefs(tree.RootNode(), &funcs)

	for _, fn := range funcs {
		tokenSpans := normalizeFunctionTokens(fn, content)
		if len(tokenSpans) < cfg.MinFuncTokens {
			continue
		}

		tokens := tokensOnly(tokenSpans)
		fp.FunctionCount++
		funcHash := hashTokens(tokens)
		fp.FunctionHashes[funcHash] = struct{}{}

		funcWindows := make(map[string]struct{})
		for windowHash, spans := range windowHashes(tokenSpans, cfg.WindowSize) {
			fp.WindowHashes[windowHash] = struct{}{}
			funcWindows[windowHash] = struct{}{}
			fp.WindowSpans[windowHash] = append(fp.WindowSpans[windowHash], spans...)
		}
		fp.FunctionWindows = append(fp.FunctionWindows, funcWindows)
	}

	return fp, nil
}

func CompareFiles(fileA, fileB string, fpA, fpB domain.FileFingerprint, cfg domain.CompareConfig) domain.PlagiarismResult {
	exactMatches := countIntersection(fpA.FunctionHashes, fpB.FunctionHashes)
	windowMatches := countIntersection(fpA.WindowHashes, fpB.WindowHashes)
	windowUnion := unionCount(fpA.WindowHashes, fpB.WindowHashes)
	perFuncScore := avgBestFunctionSimilarity(fpA.FunctionWindows, fpB.FunctionWindows)

	score := jaccard(windowMatches, windowUnion)

	matches := extractWindowMatches(fpA, fpB)

	return domain.PlagiarismResult{
		FileA:             fileA,
		FileB:             fileB,
		FunctionCountA:    fpA.FunctionCount,
		FunctionCountB:    fpB.FunctionCount,
		ExactMatches:      exactMatches,
		WindowMatches:     windowMatches,
		WindowUnion:       windowUnion,
		WindowJaccard:     score,
		PerFuncSimilarity: perFuncScore,
		Flagged:           score >= cfg.ScoreThreshold,
		Matches:           matches,
	}
}

func collectFunctionDefs(node *sitter.Node, out *[]*sitter.Node) {
	if node == nil {
		return
	}

	if node.Type() == "function_definition" {
		*out = append(*out, node)
		return
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		collectFunctionDefs(node.Child(i), out)
	}
}

func normalizeFunctionTokens(node *sitter.Node, content []byte) []domain.TokenSpan {
	idMap := make(map[string]int)
	counter := 0
	var tokens []domain.TokenSpan
	normalizeTokens(node, content, idMap, &counter, &tokens)
	return tokens
}

func normalizeTokens(node *sitter.Node, content []byte, idMap map[string]int, counter *int, tokens *[]domain.TokenSpan) {
	if node == nil {
		return
	}

	if node.Type() == "comment" {
		return
	}

	if node.ChildCount() == 0 {
		token := normalizeToken(node, content, idMap, counter)
		if token != "" {
			*tokens = append(*tokens, domain.TokenSpan{
				Token: token,
				Start: node.StartByte(),
				End:   node.EndByte(),
			})
		}
		return
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		normalizeTokens(node.Child(i), content, idMap, counter, tokens)
	}
}

func normalizeToken(node *sitter.Node, content []byte, idMap map[string]int, counter *int) string {
	raw := strings.TrimSpace(node.Content(content))
	if raw == "" {
		return ""
	}

	if !node.IsNamed() && isPunctuation(raw) {
		return ""
	}

	switch node.Type() {
	case "identifier", "field_identifier", "type_identifier":
		if id, ok := idMap[raw]; ok {
			return fmt.Sprintf("@%d", id)
		}
		*counter++
		idMap[raw] = *counter
		return fmt.Sprintf("@%d", *counter)
	case "number_literal":
		return "@NUM"
	case "string_literal":
		return "@STR"
	case "char_literal":
		return "@CHAR"
	default:
		if token, ok := normalizeOperator(raw); ok {
			return token
		}
		return raw
	}
}

func hashTokens(tokens []string) string {
	sum := sha256.Sum256([]byte(strings.Join(tokens, " ")))
	return hex.EncodeToString(sum[:])
}

func windowHashes(tokens []domain.TokenSpan, window int) map[string][]domain.Span {
	if window <= 0 || len(tokens) < window {
		return nil
	}

	hashes := make(map[string][]domain.Span, len(tokens)-window+1)
	for i := 0; i+window <= len(tokens); i++ {
		windowTokens := tokens[i : i+window]
		hash := hashTokens(tokensOnly(windowTokens))
		hashes[hash] = append(hashes[hash], domain.Span{
			Start: windowTokens[0].Start,
			End:   windowTokens[len(windowTokens)-1].End,
		})
	}
	return hashes
}

func countIntersection(a, b map[string]struct{}) int {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	if len(a) > len(b) {
		a, b = b, a
	}

	count := 0
	for k := range a {
		if _, ok := b[k]; ok {
			count++
		}
	}
	return count
}

func unionCount(a, b map[string]struct{}) int {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	seen := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}
	return len(seen)
}

func jaccard(intersection, union int) float64 {
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func avgBestFunctionSimilarity(a, b []map[string]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	avgAB := avgBestMatch(a, b)
	avgBA := avgBestMatch(b, a)
	if avgAB == 0 {
		return avgBA
	}
	if avgBA == 0 {
		return avgAB
	}
	return (avgAB + avgBA) / 2
}

func avgBestMatch(a, b []map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	sum := 0.0
	for _, fa := range a {
		best := 0.0
		for _, fb := range b {
			inter := countIntersection(fa, fb)
			union := unionCount(fa, fb)
			sim := jaccard(inter, union)
			if sim > best {
				best = sim
			}
		}
		sum += best
	}
	return sum / float64(len(a))
}

func isPunctuation(raw string) bool {
	switch raw {
	case "(", ")", "{", "}", "[", "]", ";", ",", ".":
		return true
	default:
		return false
	}
}

func normalizeOperator(raw string) (string, bool) {
	switch raw {
	case "=", "+=", "-=", "*=", "/=", "%=", "<<=", ">>=", "&=", "|=", "^=":
		return "@ASSIGN", true
	case "++", "--":
		return "@INCDEC", true
	case "+", "-", "*", "/", "%":
		return "@ARITH", true
	case "==", "!=", "<", "<=", ">", ">=":
		return "@CMP", true
	case "&&", "||", "!":
		return "@LOGIC", true
	case "&", "|", "^", "~", "<<", ">>":
		return "@BIT", true
	case "?", ":":
		return "@TERNARY", true
	default:
		return "", false
	}
}

func tokensOnly(tokens []domain.TokenSpan) []string {
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, t.Token)
	}
	return out
}

func buildLineOffsets(content []byte) []int {
	offsets := []int{0}
	for i, b := range content {
		if b == '\n' {
			offsets = append(offsets, i+1)
		}
	}
	return offsets
}

func byteToLineCol(lineOffsets []int, pos int) (int, int) {
	line := sort.Search(len(lineOffsets), func(i int) bool {
		return lineOffsets[i] > pos
	}) - 1
	if line < 0 {
		line = 0
	}
	col := pos - lineOffsets[line]
	return line + 1, col + 1
}

func extractSnippet(content []byte, start, end, max int) string {
	if start < 0 {
		start = 0
	}
	if end > len(content) {
		end = len(content)
	}
	if start >= end {
		return ""
	}
	raw := strings.ReplaceAll(string(content[start:end]), "\n", " ")
	raw = strings.TrimSpace(raw)
	if len(raw) <= max {
		return raw
	}
	return raw[:max-3] + "..."
}

func extractWindowMatches(fpA, fpB domain.FileFingerprint) []domain.WindowMatch {
	matches := make([]string, 0, len(fpA.WindowHashes))
	for k := range fpA.WindowHashes {
		if _, ok := fpB.WindowHashes[k]; ok {
			matches = append(matches, k)
		}
	}
	if len(matches) == 0 {
		return nil
	}
	sort.Strings(matches)

	result := make([]domain.WindowMatch, 0, len(matches))
	for _, hash := range matches {
		mergedA := mergeSpans(fpA.WindowSpans[hash])
		mergedB := mergeSpans(fpB.WindowSpans[hash])
		spansA := convertSpans(fpA, mergedA)
		spansB := convertSpans(fpB, mergedB)
		result = append(result, domain.WindowMatch{
			Hash:   hash[:8], // First 8 chars for display
			SpansA: spansA,
			SpansB: spansB,
		})
	}
	return result
}

func mergeSpans(spans []domain.Span) []domain.Span {
	if len(spans) <= 1 {
		return spans
	}

	cp := make([]domain.Span, len(spans))
	copy(cp, spans)
	sort.Slice(cp, func(i, j int) bool { return cp[i].Start < cp[j].Start })

	out := make([]domain.Span, 0, len(cp))
	cur := cp[0]
	for i := 1; i < len(cp); i++ {
		next := cp[i]
		if next.Start <= cur.End {
			if next.End > cur.End {
				cur.End = next.End
			}
			continue
		}
		out = append(out, cur)
		cur = next
	}
	out = append(out, cur)
	return out
}

func convertSpans(fp domain.FileFingerprint, spans []domain.Span) []domain.MatchSpan {
	result := make([]domain.MatchSpan, 0, len(spans))
	for _, sp := range spans {
		startLine, startCol := byteToLineCol(fp.LineOffsets, int(sp.Start))
		endLine, endCol := byteToLineCol(fp.LineOffsets, int(sp.End))
		snippet := extractSnippet(fp.Content, int(sp.Start), int(sp.End), 80)
		result = append(result, domain.MatchSpan{
			StartLine: startLine,
			StartCol:  startCol,
			EndLine:   endLine,
			EndCol:    endCol,
			Snippet:   snippet,
		})
	}
	return result
}

func ComputeSimilarityForProcess(submissions []domain.Submission, srcFile string, cfg domain.CompareConfig) ([]domain.SimilarityPairResult, error) {
	type fingerprintItem struct {
		idx int
		fp  domain.FileFingerprint
	}

	var fps []fingerprintItem
	var failures int

	for i, sub := range submissions {
		path := filepath.Join(sub.Path, srcFile)
		fp, err := FingerprintFile(path, cfg)
		if err != nil {
			failures++
			continue
		}
		fps = append(fps, fingerprintItem{idx: i, fp: fp})
	}

	if len(fps) < 2 {
		if failures > 0 && len(fps) == 0 {
			return nil, fmt.Errorf("no fingerprints created (%d failures). check source file selection and parseability", failures)
		}
		return []domain.SimilarityPairResult{}, nil
	}

	var pairs []domain.SimilarityPairResult
	for i := 0; i < len(fps); i++ {
		for j := i + 1; j < len(fps); j++ {
			a := fps[i]
			b := fps[j]
			res := CompareFiles(
				filepath.Base(submissions[a.idx].ID),
				filepath.Base(submissions[b.idx].ID),
				a.fp,
				b.fp,
				cfg,
			)
			pairs = append(pairs, domain.SimilarityPairResult{
				AIndex: a.idx,
				BIndex: b.idx,
				Result: res,
			})
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Result.WindowJaccard > pairs[j].Result.WindowJaccard
	})

	return pairs, nil
}
