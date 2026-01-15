package atlas

import (
	"context"

	"github.com/felipetrejos/felituive/internal/ports"
)

// Retriever implements ports.Retriever using MongoDB Atlas Vector Search
type Retriever struct {
	// client *mongo.Client
}

// New creates a new Atlas Vector Search retriever
func New(uri string) (*Retriever, error) {
	// TODO: Initialize MongoDB connection
	return &Retriever{}, nil
}

func (r *Retriever) Search(ctx context.Context, corpusID string, query []float32, limit int) ([]ports.SearchResult, error) {
	// TODO: Implement Atlas Vector Search
	return nil, nil
}

// Ensure Retriever implements ports.Retriever
var _ ports.Retriever = (*Retriever)(nil)
