package ports

import (
	"context"

	"github.com/felipetrejos/felituive/internal/domain"
)

// LLM defines the interface for language model interactions
type LLM interface {
	// Chat sends messages and returns a response
	Chat(ctx context.Context, messages []domain.Message) (string, error)

	// ChatStream sends messages and streams the response
	ChatStream(ctx context.Context, messages []domain.Message, onChunk func(chunk string)) error
}
