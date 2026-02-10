package llm

import "testing"

func TestUserMessage(t *testing.T) {
	msg := NewUserMessage("hello")
	if msg.Type() != TypeUserMessage {
		t.Errorf("expected type %s, got %s", TypeUserMessage, msg.Type())
	}
}

func TestAssistantTextMessage(t *testing.T) {
	msg := NewAssistantTextMessage("thinking")
	if msg.Type() != TypeAssistantText {
		t.Errorf("expected type %s", TypeAssistantText)
	}
}

func TestAssistantActionMessage(t *testing.T) {
	msg := NewAssistantActionMessage("fs.read", map[string]any{"path": "file.txt"})
	if msg.Type() != TypeAssistantAction {
		t.Errorf("expected type %s", TypeAssistantAction)
	}
}

func TestToolResultMessage(t *testing.T) {
	msg := NewToolResultMessage("id1", "fs.read", "data")
	if msg.Type() != TypeToolResult {
		t.Errorf("expected type %s", TypeToolResult)
	}
}

func TestToolErrorMessage(t *testing.T) {
	msg := NewToolErrorMessage("id1", "fs.read", "error")
	if msg.Type() != TypeToolError {
		t.Errorf("expected type %s", TypeToolError)
	}
}

func TestSystemContextMessage(t *testing.T) {
	msg := NewSystemContextMessage("system prompt")
	if msg.Type() != TypeSystemMessage {
		t.Errorf("expected type %s", TypeSystemMessage)
	}
}

func TestConversation(t *testing.T) {
	conv := NewConversation()
	if conv.Length() != 0 {
		t.Error("expected empty conversation")
	}

	conv.Add(NewUserMessage("hello"), "received", nil)
	if conv.Length() != 1 {
		t.Error("expected length 1")
	}

	entries := conv.GetAll()
	if len(entries) != 1 {
		t.Error("expected 1 entry")
	}
}
