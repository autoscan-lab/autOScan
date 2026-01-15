package services

import (
	"context"

	"github.com/felipetrejos/felituive/internal/domain"
	"github.com/felipetrejos/felituive/internal/ports"
)

// Indexer handles document indexing operations
type Indexer struct {
	storage  ports.Storage
	embedder ports.Embedder
}

// NewIndexer creates a new indexer service
func NewIndexer(storage ports.Storage, embedder ports.Embedder) *Indexer {
	return &Indexer{
		storage:  storage,
		embedder: embedder,
	}
}

// IndexFolder indexes all files in a folder into a corpus
func (i *Indexer) IndexFolder(ctx context.Context, corpusName, path string) (*domain.Corpus, error) {
	// TODO: Implement
	// 1. Create or get corpus
	// 2. Walk directory and find files
	// 3. Chunk each file
	// 4. Generate embeddings
	// 5. Save chunks
	return nil, nil
}

// ReindexCorpus re-indexes an existing corpus
func (i *Indexer) ReindexCorpus(ctx context.Context, corpusID string) error {
	// TODO: Implement
	return nil
}
