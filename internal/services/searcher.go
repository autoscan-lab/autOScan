package services

import (
	"context"

	"github.com/felipetrejos/felituive/internal/ports"
)

// Searcher handles semantic search operations
type Searcher struct {
	embedder  ports.Embedder
	retriever ports.Retriever
}

// NewSearcher creates a new searcher service
func NewSearcher(embedder ports.Embedder, retriever ports.Retriever) *Searcher {
	return &Searcher{
		embedder:  embedder,
		retriever: retriever,
	}
}

// Search performs semantic search on a corpus
func (s *Searcher) Search(ctx context.Context, corpusID, query string, limit int) ([]ports.SearchResult, error) {
	// Embed the query
	embedding, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	// Retrieve similar chunks
	return s.retriever.Search(ctx, corpusID, embedding, limit)
}
