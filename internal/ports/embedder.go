package ports

import "context"

// Embedder defines the interface for generating text embeddings
type Embedder interface {
	// Embed generates an embedding vector for the given text
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates embeddings for multiple texts
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimensions returns the embedding vector size
	Dimensions() int
}
