package session

import (
	"context"
	"testing"

	"github.com/cshaiku/goshi/internal/llm"
)

// MockBackend implements llm.Backend for testing
type MockBackend struct {
	Responses []string // Allow customization of responses
	CallCount int      // Track call count
}

func (m *MockBackend) Stream(ctx context.Context, system string, messages []llm.Message) (llm.Stream, error) {
	data := m.Responses
	if data == nil {
		data = []string{"test response"}
	}
	m.CallCount++
	return &MockStream{Index: 0, Data: data}, nil
}

// MockStream implements llm.Stream for testing
type MockStream struct {
	Index int
	Data  []string
}

func (m *MockStream) Recv() (string, error) {
	if m.Index >= len(m.Data) {
		return "", &MockStreamError{}
	}
	chunk := m.Data[m.Index]
	m.Index++
	return chunk, nil
}

func (m *MockStream) Close() error {
	return nil
}

type MockStreamError struct{}

func (e *MockStreamError) Error() string   { return "end of stream" }
func (e *MockStreamError) Timeout() bool   { return false }
func (e *MockStreamError) Temporary() bool { return false }

func TestNewChatSession(t *testing.T) {
	ctx := context.Background()
	backend := &MockBackend{}

	session, err := NewChatSession(ctx, "test system prompt", backend)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if session == nil {
		t.Fatal("session is nil")
	}

	if session.SystemPrompt != "test system prompt" {
		t.Errorf("expected system prompt 'test system prompt', got '%s'", session.SystemPrompt)
	}

	if session.Permissions == nil {
		t.Fatal("permissions is nil")
	}

	if len(session.Messages) != 0 {
		t.Errorf("expected empty messages, got %d messages", len(session.Messages))
	}
}

func TestChatSession_AddUserMessage(t *testing.T) {
	ctx := context.Background()
	session, _ := NewChatSession(ctx, "test", &MockBackend{})

	session.AddUserMessage("Hello bot")

	if len(session.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(session.Messages))
	}

	userMsg, ok := session.Messages[0].(*llm.UserMessage)
	if !ok {
		t.Fatalf("expected UserMessage type, got %T", session.Messages[0])
	}

	if userMsg.Content != "Hello bot" {
		t.Errorf("expected content 'Hello bot', got '%s'", userMsg.Content)
	}
}

func TestChatSession_GrantPermission(t *testing.T) {
	ctx := context.Background()
	session, _ := NewChatSession(ctx, "test", &MockBackend{})

	if session.HasPermission("FS_READ") {
		t.Fatal("FS_READ should not be granted initially")
	}

	session.GrantPermission("FS_READ")

	if !session.HasPermission("FS_READ") {
		t.Fatal("FS_READ should be granted after GrantPermission")
	}

	if len(session.Permissions.AuditLog) != 1 {
		t.Errorf("expected 1 audit log entry, got %d", len(session.Permissions.AuditLog))
	}

	entry := session.Permissions.AuditLog[0]
	if entry.Action != "GRANT" || entry.Capability != "FS_READ" {
		t.Errorf("expected GRANT FS_READ, got %s %s", entry.Action, entry.Capability)
	}
}

func TestChatSession_DenyPermission(t *testing.T) {
	ctx := context.Background()
	session, _ := NewChatSession(ctx, "test", &MockBackend{})

	session.DenyPermission("FS_WRITE")

	if len(session.Permissions.AuditLog) != 1 {
		t.Errorf("expected 1 audit log entry, got %d", len(session.Permissions.AuditLog))
	}

	entry := session.Permissions.AuditLog[0]
	if entry.Action != "DENY" || entry.Capability != "FS_WRITE" {
		t.Errorf("expected DENY FS_WRITE, got %s %s", entry.Action, entry.Capability)
	}
}

func TestChatSession_GetAuditLog(t *testing.T) {
	ctx := context.Background()
	session, _ := NewChatSession(ctx, "test", &MockBackend{})

	session.GrantPermission("FS_READ")
	session.DenyPermission("FS_WRITE")

	auditLog := session.GetAuditLog()

	if auditLog == "" {
		t.Error("audit log should not be empty")
	}

	if len(session.Permissions.AuditLog) != 2 {
		t.Errorf("expected 2 audit entries, got %d", len(session.Permissions.AuditLog))
	}
}
