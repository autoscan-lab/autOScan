package domain

// Role represents the sender of a message
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// Message represents a chat message
type Message struct {
	Role    Role
	Content string
}

// ChatSession represents a conversation with context
type ChatSession struct {
	ID       string
	CorpusID string
	Messages []Message
}
