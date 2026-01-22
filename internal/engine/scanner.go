package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/felipetrejos/autoscan/internal/domain"
	"github.com/felipetrejos/autoscan/internal/policy"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/c"
)

// ScanEngine scans source files for banned function calls using tree-sitter.
type ScanEngine struct {
	policy    *policy.Policy
	bannedSet map[string]struct{}
	parser    *sitter.Parser
	lang      *sitter.Language
}

// NewScanEngine creates a new scan engine.
func NewScanEngine(p *policy.Policy) *ScanEngine {
	parser := sitter.NewParser()
	lang := c.GetLanguage()
	parser.SetLanguage(lang)

	return &ScanEngine{
		policy:    p,
		bannedSet: p.BannedSet(),
		parser:    parser,
		lang:      lang,
	}
}

// Scan scans a submission for banned function calls.
func (e *ScanEngine) Scan(sub domain.Submission) domain.ScanResult {
	var allHits []domain.BannedHit
	var parseErrors []string

	for _, cFile := range sub.CFiles {
		filePath := filepath.Join(sub.Path, cFile)
		hits, err := e.scanFile(filePath, cFile)
		if err != nil {
			parseErrors = append(parseErrors, cFile+": "+err.Error())
			continue
		}
		allHits = append(allHits, hits...)
	}

	return domain.NewScanResult(allHits, parseErrors)
}

// ScanAll scans all submissions.
func (e *ScanEngine) ScanAll(submissions []domain.Submission, onComplete func(domain.Submission, domain.ScanResult)) []domain.ScanResult {
	results := make([]domain.ScanResult, len(submissions))

	for i, sub := range submissions {
		result := e.Scan(sub)
		results[i] = result
		if onComplete != nil {
			onComplete(sub, result)
		}
	}

	return results
}

func (e *ScanEngine) scanFile(filePath, displayName string) ([]domain.BannedHit, error) {
	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Parse with tree-sitter
	tree, err := e.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	// Extract function calls
	var hits []domain.BannedHit
	lines := strings.Split(string(content), "\n")

	e.walkTree(tree.RootNode(), content, lines, displayName, &hits)

	return hits, nil
}

func (e *ScanEngine) walkTree(node *sitter.Node, content []byte, lines []string, fileName string, hits *[]domain.BannedHit) {
	if node == nil {
		return
	}

	// Check if this is a call expression
	if node.Type() == "call_expression" {
		e.checkCallExpression(node, content, lines, fileName, hits)
	}

	// Recurse into children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		e.walkTree(child, content, lines, fileName, hits)
	}
}

func (e *ScanEngine) checkCallExpression(node *sitter.Node, content []byte, lines []string, fileName string, hits *[]domain.BannedHit) {
	// Get the function being called (first child is usually the function identifier)
	if node.ChildCount() == 0 {
		return
	}

	funcNode := node.Child(0)
	if funcNode == nil {
		return
	}

	var funcName string

	switch funcNode.Type() {
	case "identifier":
		// Direct function call: printf(...)
		funcName = funcNode.Content(content)
	case "field_expression":
		// Member access: obj.method(...) - get the field name
		if funcNode.ChildCount() >= 3 {
			field := funcNode.Child(2) // Usually: object . field
			if field != nil && field.Type() == "field_identifier" {
				funcName = field.Content(content)
			}
		}
	default:
		// Other cases (function pointers, etc.) - skip for now
		return
	}

	if funcName == "" {
		return
	}

	// Check if this function is banned
	if _, banned := e.bannedSet[funcName]; banned {
		line := int(funcNode.StartPoint().Row) + 1 // 1-based
		col := int(funcNode.StartPoint().Column) + 1

		// Get snippet (the line containing the call)
		snippet := ""
		if line-1 < len(lines) {
			snippet = strings.TrimSpace(lines[line-1])
			// Truncate long snippets
			if len(snippet) > 80 {
				snippet = snippet[:77] + "..."
			}
		}

		*hits = append(*hits, domain.NewBannedHit(
			funcName,
			fileName,
			line,
			col,
			snippet,
		))
	}
}
