package ports

import (
	"context"

	"github.com/felipetrejos/felituive/internal/domain"
)

// SearchResult represents a chunk with its relevance score
type SearchResult struct {
	Chunk *domain.Chunk
	Score float64
}

// Retriever defines the interface for semantic search
type Retriever interface {
	// Search finds the most relevant chunks for a query
	Search(ctx context.Context, corpusID string, query []float32, limit int) ([]SearchResult, error)
}
