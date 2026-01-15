package sqlite

import (
	"context"

	"github.com/felipetrejos/felituive/internal/domain"
	"github.com/felipetrejos/felituive/internal/ports"
)

// Store implements ports.Storage using SQLite
type Store struct {
	// db *sql.DB
}

// New creates a new SQLite store
func New(path string) (*Store, error) {
	// TODO: Initialize SQLite connection
	return &Store{}, nil
}

func (s *Store) CreateCorpus(ctx context.Context, corpus *domain.Corpus) error {
	// TODO: Implement
	return nil
}

func (s *Store) GetCorpus(ctx context.Context, id string) (*domain.Corpus, error) {
	// TODO: Implement
	return nil, nil
}

func (s *Store) GetCorpusByName(ctx context.Context, name string) (*domain.Corpus, error) {
	// TODO: Implement
	return nil, nil
}

func (s *Store) ListCorpora(ctx context.Context) ([]*domain.Corpus, error) {
	// TODO: Implement
	return nil, nil
}

func (s *Store) DeleteCorpus(ctx context.Context, id string) error {
	// TODO: Implement
	return nil
}

func (s *Store) SaveChunks(ctx context.Context, chunks []*domain.Chunk) error {
	// TODO: Implement
	return nil
}

func (s *Store) GetChunksByCorpus(ctx context.Context, corpusID string) ([]*domain.Chunk, error) {
	// TODO: Implement
	return nil, nil
}

func (s *Store) DeleteChunksByCorpus(ctx context.Context, corpusID string) error {
	// TODO: Implement
	return nil
}

func (s *Store) Close() error {
	// TODO: Implement
	return nil
}

// Ensure Store implements ports.Storage
var _ ports.Storage = (*Store)(nil)
