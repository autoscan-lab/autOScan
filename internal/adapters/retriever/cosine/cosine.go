package cosine

import (
	"context"

	"github.com/felipetrejos/felituive/internal/ports"
)

// Retriever implements ports.Retriever using cosine similarity
type Retriever struct {
	storage ports.Storage
}

// New creates a new cosine similarity retriever
func New(storage ports.Storage) *Retriever {
	return &Retriever{storage: storage}
}

func (r *Retriever) Search(ctx context.Context, corpusID string, query []float32, limit int) ([]ports.SearchResult, error) {
	// TODO: Implement cosine similarity search
	// 1. Get all chunks for corpus
	// 2. Calculate cosine similarity for each
	// 3. Sort by score and return top N
	return nil, nil
}

// Ensure Retriever implements ports.Retriever
var _ ports.Retriever = (*Retriever)(nil)
