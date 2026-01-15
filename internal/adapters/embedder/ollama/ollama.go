package ollama

import (
	"context"

	"github.com/felipetrejos/felituive/internal/ports"
)

// Embedder implements ports.Embedder using Ollama
type Embedder struct {
	model      string
	dimensions int
}

// New creates a new Ollama embedder
func New(model string) *Embedder {
	return &Embedder{
		model:      model,
		dimensions: 384, // Default for nomic-embed-text
	}
}

func (e *Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
	// TODO: Implement Ollama embedding API call
	return nil, nil
}

func (e *Embedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	// TODO: Implement batch embedding
	return nil, nil
}

func (e *Embedder) Dimensions() int {
	return e.dimensions
}

// Ensure Embedder implements ports.Embedder
var _ ports.Embedder = (*Embedder)(nil)
