package ports

import (
	"context"

	"github.com/felipetrejos/felituive/internal/domain"
)

// Storage defines the interface for corpus and chunk persistence
type Storage interface {
	// Corpus operations
	CreateCorpus(ctx context.Context, corpus *domain.Corpus) error
	GetCorpus(ctx context.Context, id string) (*domain.Corpus, error)
	GetCorpusByName(ctx context.Context, name string) (*domain.Corpus, error)
	ListCorpora(ctx context.Context) ([]*domain.Corpus, error)
	DeleteCorpus(ctx context.Context, id string) error

	// Chunk operations
	SaveChunks(ctx context.Context, chunks []*domain.Chunk) error
	GetChunksByCorpus(ctx context.Context, corpusID string) ([]*domain.Chunk, error)
	DeleteChunksByCorpus(ctx context.Context, corpusID string) error

	// Lifecycle
	Close() error
}
