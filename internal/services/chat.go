package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/felipetrejos/felituive/internal/domain"
	"github.com/felipetrejos/felituive/internal/ports"
)

// Chat handles RAG chat operations
type Chat struct {
	llm      ports.LLM
	searcher *Searcher
}

// NewChat creates a new chat service
func NewChat(llm ports.LLM, searcher *Searcher) *Chat {
	return &Chat{
		llm:      llm,
		searcher: searcher,
	}
}

// Send sends a message with RAG context and returns the response
func (c *Chat) Send(ctx context.Context, corpusID, userMessage string) (string, []ports.SearchResult, error) {
	// Search for relevant context
	results, err := c.searcher.Search(ctx, corpusID, userMessage, 5)
	if err != nil {
		return "", nil, err
	}

	// Build context-enhanced prompt
	messages := c.buildMessages(userMessage, results)

	// Get LLM response
	response, err := c.llm.Chat(ctx, messages)
	if err != nil {
		return "", nil, err
	}

	return response, results, nil
}

// SendStream sends a message and streams the response
func (c *Chat) SendStream(ctx context.Context, corpusID, userMessage string, onChunk func(chunk string)) ([]ports.SearchResult, error) {
	// Search for relevant context
	results, err := c.searcher.Search(ctx, corpusID, userMessage, 5)
	if err != nil {
		return nil, err
	}

	// Build context-enhanced prompt
	messages := c.buildMessages(userMessage, results)

	// Stream LLM response
	if err := c.llm.ChatStream(ctx, messages, onChunk); err != nil {
		return nil, err
	}

	return results, nil
}

func (c *Chat) buildMessages(userMessage string, results []ports.SearchResult) []domain.Message {
	var contextParts []string
	for _, r := range results {
		contextParts = append(contextParts, fmt.Sprintf("[%s]\n%s", r.Chunk.FilePath, r.Chunk.Content))
	}

	systemPrompt := fmt.Sprintf(`You are a helpful assistant. Answer questions based on the following context:

%s

If the context doesn't contain relevant information, say so.`, strings.Join(contextParts, "\n\n---\n\n"))

	return []domain.Message{
		{Role: domain.RoleSystem, Content: systemPrompt},
		{Role: domain.RoleUser, Content: userMessage},
	}
}
