package llm

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MessageType discriminates the purpose and role of a message
type MessageType string

const (
	TypeUserMessage     MessageType = "user"             // User input to the assistant
	TypeAssistantText   MessageType = "assistant_text"   // Planning or reasoning from LLM
	TypeAssistantAction MessageType = "assistant_action" // Tool call request from LLM
	TypeToolResult      MessageType = "tool_result"      // Successful tool execution result
	TypeToolError       MessageType = "tool_error"       // Tool execution error
	TypeSystemMessage   MessageType = "system_context"   // System instructions or self-model
)

// LLMMessage is the interface that all structured messages implement
type LLMMessage interface {
	Type() MessageType
	ToAPIFormat() map[string]string // For OpenAI/Ollama API calls
	ToLog() map[string]any          // For audit logging
}

// UserMessage represents user input
type UserMessage struct {
	Content string
	ID      string
}

func NewUserMessage(content string) *UserMessage {
	return &UserMessage{
		Content: content,
		ID:      uuid.New().String(),
	}
}

func (m *UserMessage) Type() MessageType {
	return TypeUserMessage
}

func (m *UserMessage) ToAPIFormat() map[string]string {
	return map[string]string{
		"role":    "user",
		"content": m.Content,
	}
}

func (m *UserMessage) ToLog() map[string]any {
	return map[string]any{
		"type":    m.Type(),
		"id":      m.ID,
		"content": m.Content,
	}
}

// AssistantTextMessage represents planning or reasoning from the LLM
// (not a tool call, just thinking/planning)
type AssistantTextMessage struct {
	Content string
	ID      string
}

func NewAssistantTextMessage(content string) *AssistantTextMessage {
	return &AssistantTextMessage{
		Content: content,
		ID:      uuid.New().String(),
	}
}

func (m *AssistantTextMessage) Type() MessageType {
	return TypeAssistantText
}

func (m *AssistantTextMessage) ToAPIFormat() map[string]string {
	return map[string]string{
		"role":    "assistant",
		"content": m.Content,
	}
}

func (m *AssistantTextMessage) ToLog() map[string]any {
	return map[string]any{
		"type":    m.Type(),
		"id":      m.ID,
		"content": m.Content,
	}
}

// AssistantActionMessage represents a tool call requested by the LLM
type AssistantActionMessage struct {
	ToolName string
	ToolArgs map[string]any
	ToolID   string // For matching with result
	ID       string
}

func NewAssistantActionMessage(toolName string, toolArgs map[string]any) *AssistantActionMessage {
	return &AssistantActionMessage{
		ToolName: toolName,
		ToolArgs: toolArgs,
		ToolID:   uuid.New().String(),
		ID:       uuid.New().String(),
	}
}

func (m *AssistantActionMessage) Type() MessageType {
	return TypeAssistantAction
}

func (m *AssistantActionMessage) ToAPIFormat() map[string]string {
	argsJSON, _ := json.Marshal(m.ToolArgs)
	return map[string]string{
		"role": "assistant",
		"content": fmt.Sprintf(
			"Calling tool '%s' with arguments: %s",
			m.ToolName, string(argsJSON),
		),
	}
}

func (m *AssistantActionMessage) ToLog() map[string]any {
	return map[string]any{
		"type":     m.Type(),
		"id":       m.ID,
		"toolId":   m.ToolID,
		"toolName": m.ToolName,
		"toolArgs": m.ToolArgs,
	}
}

// ToolResultMessage represents the result of a tool execution
type ToolResultMessage struct {
	ToolID     string
	ToolName   string
	Success    bool
	Result     any    // Successful result
	Error      string // Error message if !Success
	ExecutedAt time.Time
	ID         string
}

func NewToolResultMessage(toolID string, toolName string, result any) *ToolResultMessage {
	return &ToolResultMessage{
		ToolID:     toolID,
		ToolName:   toolName,
		Success:    true,
		Result:     result,
		Error:      "",
		ExecutedAt: time.Now(),
		ID:         uuid.New().String(),
	}
}

func NewToolErrorMessage(toolID string, toolName string, err string) *ToolResultMessage {
	return &ToolResultMessage{
		ToolID:     toolID,
		ToolName:   toolName,
		Success:    false,
		Result:     nil,
		Error:      err,
		ExecutedAt: time.Now(),
		ID:         uuid.New().String(),
	}
}

func (m *ToolResultMessage) Type() MessageType {
	if m.Success {
		return TypeToolResult
	}
	return TypeToolError
}

func (m *ToolResultMessage) ToAPIFormat() map[string]string {
	content := fmt.Sprintf("Tool %s result: %v", m.ToolName, m.Result)
	if !m.Success {
		content = fmt.Sprintf("Tool %s error: %s", m.ToolName, m.Error)
	}
	return map[string]string{
		"role":    "user",
		"content": content,
	}
}

func (m *ToolResultMessage) ToLog() map[string]any {
	msgType := "tool_result"
	if !m.Success {
		msgType = "tool_error"
	}
	return map[string]any{
		"type":       msgType,
		"id":         m.ID,
		"toolId":     m.ToolID,
		"toolName":   m.ToolName,
		"success":    m.Success,
		"result":     m.Result,
		"error":      m.Error,
		"executedAt": m.ExecutedAt,
	}
}

// SystemContextMessage represents system instructions or self-model context
type SystemContextMessage struct {
	Content string
	ID      string
}

func NewSystemContextMessage(content string) *SystemContextMessage {
	return &SystemContextMessage{
		Content: content,
		ID:      uuid.New().String(),
	}
}

func (m *SystemContextMessage) Type() MessageType {
	return TypeSystemMessage
}

func (m *SystemContextMessage) ToAPIFormat() map[string]string {
	return map[string]string{
		"role":    "system",
		"content": m.Content,
	}
}

func (m *SystemContextMessage) ToLog() map[string]any {
	return map[string]any{
		"type":    m.Type(),
		"id":      m.ID,
		"content": m.Content,
	}
}

// ConversationEntry represents a single entry in the conversation history
// with metadata about decisions and audit trail
type ConversationEntry struct {
	ID        string // UUID for traceability
	Timestamp time.Time
	Message   LLMMessage
	Decision  string         // e.g., "tool_call", "permission_denied", "error"
	Audit     map[string]any // Decision metadata
}

// Conversation represents a conversation history with structured entries
type Conversation struct {
	entries []ConversationEntry
}

// NewConversation creates a new empty conversation
func NewConversation() *Conversation {
	return &Conversation{
		entries: []ConversationEntry{},
	}
}

// Add appends a new message to the conversation with decision metadata
func (c *Conversation) Add(msg LLMMessage, decision string, audit map[string]any) *ConversationEntry {
	entry := ConversationEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Message:   msg,
		Decision:  decision,
		Audit:     audit,
	}
	c.entries = append(c.entries, entry)
	return &entry
}

// GetAll returns all entries in the conversation
func (c *Conversation) GetAll() []ConversationEntry {
	return c.entries
}

// GetMessages returns just the messages (without decision metadata)
// formatted for LLM API calls
func (c *Conversation) GetMessages() []Message {
	result := make([]Message, 0, len(c.entries))
	for _, entry := range c.entries {
		apiFormat := entry.Message.ToAPIFormat()
		result = append(result, Message{
			Role:    apiFormat["role"],
			Content: apiFormat["content"],
		})
	}
	return result
}

// GetAuditLog returns all entries formatted for audit logging
func (c *Conversation) GetAuditLog() []map[string]any {
	result := make([]map[string]any, 0, len(c.entries))
	for _, entry := range c.entries {
		logEntry := map[string]any{
			"entryId":   entry.ID,
			"timestamp": entry.Timestamp,
			"message":   entry.Message.ToLog(),
			"decision":  entry.Decision,
			"audit":     entry.Audit,
		}
		result = append(result, logEntry)
	}
	return result
}

// Length returns the number of entries in the conversation
func (c *Conversation) Length() int {
	return len(c.entries)
}
