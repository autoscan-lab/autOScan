package ollama

import (
	"context"

	"github.com/felipetrejos/felituive/internal/domain"
	"github.com/felipetrejos/felituive/internal/ports"
)

// Client implements ports.LLM using Ollama
type Client struct {
	model string
	host  string
}

// New creates a new Ollama LLM client
func New(model, host string) *Client {
	if host == "" {
		host = "http://localhost:11434"
	}
	return &Client{
		model: model,
		host:  host,
	}
}

func (c *Client) Chat(ctx context.Context, messages []domain.Message) (string, error) {
	// TODO: Implement Ollama chat API call
	return "", nil
}

func (c *Client) ChatStream(ctx context.Context, messages []domain.Message, onChunk func(chunk string)) error {
	// TODO: Implement streaming chat
	return nil
}

// Ensure Client implements ports.LLM
var _ ports.LLM = (*Client)(nil)
