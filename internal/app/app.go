package app

import (
	"context"

	"github.com/cshaiku/goshi/internal/llm"
)

// ChatSession encapsulates pure chat state.
// No IO, no CLI, no printing.
type ChatSession struct {
	Client   llm.Client
	Messages []llm.Message
}

// NewChatSession creates a new chat session with an initial system prompt.
func NewChatSession(client llm.Client, systemPrompt string) *ChatSession {
	msgs := []llm.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	return &ChatSession{
		Client:   client,
		Messages: msgs,
	}
}

// StreamResponse streams a single assistant response.
// The caller is responsible for:
// - displaying output
// - handling tool calls
// - appending messages
func (s *ChatSession) StreamResponse(
	ctx context.Context,
) (llm.Stream, error) {
	return s.Client.Stream(ctx, s.Messages)
}

// AppendUserMessage appends a user message.
func (s *ChatSession) AppendUserMessage(content string) {
	s.Messages = append(s.Messages, llm.Message{
		Role:    "user",
		Content: content,
	})
}

// AppendAssistantMessage appends an assistant message.
func (s *ChatSession) AppendAssistantMessage(content string) {
	s.Messages = append(s.Messages, llm.Message{
		Role:    "assistant",
		Content: content,
	})
}

// AppendSystemMessage appends a system/tool message.
func (s *ChatSession) AppendSystemMessage(content string) {
	s.Messages = append(s.Messages, llm.Message{
		Role:    "system",
		Content: content,
	})
}
