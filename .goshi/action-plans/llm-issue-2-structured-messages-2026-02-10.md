# Action Plan: Structured Message Type System

**Status:** Not Started  
**Priority:** P0 (Blocker for all phases)  
**Effort:** 1.5 days  
**Dependencies:** None (can be done in parallel with tool registry)  
**Blocks:** Phase 2, 3, 4

---

## Problem Statement

**Current Issue:** Ambiguous message types undermine conversation semantics and auditability
- `Message{Role: "user/assistant", Content: string}` is undifferentiated text
- Tool results use `Role: "system"` which conflicts with system prompt context
- No way to distinguish: planning text, action declarations, error reports, tool results
- Message history has ambiguous semantics; unclear what should be logged/audited
- Cannot easily extend for future message types (reasoning, intermediate steps, warnings)

**Risk:** Conversation replay, debugging, and audit trails become unreliable; cannot trace decisions to message sources.

---

## Solution Design

### 1. Structured Message Types (OpenAI Style)

Create `internal/llm/messages.go`:

```go
// MessageType discriminates message purpose
type MessageType string

const (
    TypeUserMessage      MessageType = "user"
    TypeAssistantText    MessageType = "assistant_text"     // Planning/reasoning
    TypeAssistantAction  MessageType = "assistant_action"   // Tool call
    TypeToolResult       MessageType = "tool_result"         // Tool execution result
    TypeToolError        MessageType = "tool_error"          // Tool execution failure
    TypeSystemMessage    MessageType = "system_context"      // Self-model, instructions
)

// LLMMessage is the interface all messages implement
type LLMMessage interface {
    Type() MessageType
    ToAPIFormat() map[string]string      // For OpenAI/Ollama API
    ToLog() map[string]any               // For audit log
}

// UserMessage: user input
type UserMessage struct {
    Content string
    ID      string  // UUID for traceability
}

func (m *UserMessage) Type() MessageType { return TypeUserMessage }
func (m *UserMessage) ToAPIFormat() map[string]string {
    return map[string]string{"role": "user", "content": m.Content}
}

// AssistantTextMessage: planning/reasoning from LLM
type AssistantTextMessage struct {
    Content string
}

func (m *AssistantTextMessage) Type() MessageType { return TypeAssistantText }
func (m *AssistantTextMessage) ToAPIFormat() map[string]string {
    return map[string]string{"role": "assistant", "content": m.Content}
}

// AssistantActionMessage: tool call from LLM
type AssistantActionMessage struct {
    ToolName string
    ToolArgs map[string]any
    ToolID   string  // For matching with result
}

func (m *AssistantActionMessage) Type() MessageType { return TypeAssistantAction }
func (m *AssistantActionMessage) ToAPIFormat() map[string]string {
    // For follow-up to LLM, serialize action as context
    jsonBytes, _ := json.Marshal(m.ToolArgs)
    return map[string]string{
        "role": "assistant",
        "content": fmt.Sprintf("I called: %s with args %s", m.ToolName, string(jsonBytes)),
    }
}

// ToolResultMessage: result of tool execution
type ToolResultMessage struct {
    ToolID    string
    ToolName  string
    Success   bool
    Result    any      // Successful result
    Error     string   // Error if !Success
    ExecutedAt time.Time
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
    return map[string]string{"role": "user", "content": content}
}

// SystemContextMessage: self-model, instructions (unchanged)
type SystemContextMessage struct {
    Content string
}

func (m *SystemContextMessage) Type() MessageType { return TypeSystemMessage }
```

### 2. Conversation History with Metadata

Create `internal/cli/conversation.go`:

```go
type ConversationEntry struct {
    ID            string           // UUID for traceability
    Timestamp     time.Time
    Message       LLMMessage
    Decision      string           // e.g., "tool_call", "permission_denied", "error"
    Audit         map[string]any   // Decision metadata
}

type Conversation struct {
    entries    []ConversationEntry
    mu         sync.RWMutex
}

func (c *Conversation) Add(msg LLMMessage, decision string, audit map[string]any) {
    entry := ConversationEntry{
        ID:        uuid.New().String(),
        Timestamp: time.Now(),
        Message:   msg,
        Decision:  decision,
        Audit:     audit,
    }
    c.mu.Lock()
    c.entries = append(c.entries, entry)
    c.mu.Unlock()
}

func (c *Conversation) Messages() []LLMMessage {
    c.mu.RLock()
    defer c.mu.RUnlock()
    msgs := make([]LLMMessage, len(c.entries))
    for i, e := range c.entries {
        msgs[i] = e.Message
    }
    return msgs
}

func (c *Conversation) AuditLog() []ConversationEntry {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return append([]ConversationEntry{}, c.entries...)
}
```

### 3. Conversation API for LLM Backend

Update `llm.Backend` interface:

```go
type Backend interface {
    // Old signature:
    // Stream(ctx context.Context, system string, messages []Message) (Stream, error)
    
    // New signature:
    StreamStructured(
        ctx context.Context,
        systemContext *SystemContextMessage,
        history []LLMMessage,
    ) (Stream, error)
}
```

Backend converts to API format:

```go
func (c *Client) StreamStructured(
    ctx context.Context,
    system *SystemContextMessage,
    history []LLMMessage,
) (Stream, error) {
    apiMessages := []map[string]string{}
    
    // Add system context
    apiMessages = append(apiMessages, system.ToAPIFormat())
    
    // Add history, filtering system messages
    for _, msg := range history {
        if msg.Type() != TypeSystemMessage {
            apiMessages = append(apiMessages, msg.ToAPIFormat())
        }
    }
    
    // ... rest of request building
}
```

### 4. Message Validation

```go
func ValidateMessage(msg LLMMessage) error {
    switch m := msg.(type) {
    case *UserMessage:
        if strings.TrimSpace(m.Content) == "" {
            return errors.New("user message cannot be empty")
        }
    case *AssistantActionMessage:
        if m.ToolName == "" {
            return errors.New("action must specify tool name")
        }
        if m.ToolArgs == nil {
            m.ToolArgs = make(map[string]any)
        }
    case *ToolResultMessage:
        if m.ToolID == "" {
            return errors.New("tool result must reference tool ID")
        }
    }
    return nil
}
```

---

## Implementation Steps

1. **Define message types** (`llm/messages.go`)
   - Create interfaces and concrete types
   - Implement ToAPIFormat() and ToLog() for each

2. **Create conversation history** (`cli/conversation.go`)
   - Conversation struct with add/query methods
   - Entry tracing with IDs and timestamps
   - Audit log export

3. **Update LLM backend** (`llm/backend.go` and `ollama/client.go`)
   - Change Stream() signature to StreamStructured()
   - Convert message types to API format
   - Maintain backwards compatibility with adapter if needed

4. **Update chat loop** (`cli/chat.go`)
   - Replace `messages := []Message{}` with conversation history
   - Use typed messages when creating entries
   - Log decisions to audit trail

5. **Tests** (`llm/messages_test.go`, `cli/conversation_test.go`)
   - Test each message type serialization
   - Test conversation tracking and audit log
   - Test message validation
   - Test API format conversion

---

## Audit Log Example

```json
{
  "entries": [
    {
      "id": "uuid-1",
      "timestamp": "2026-02-10T15:30:00Z",
      "message": {"type": "user", "content": "read the README"},
      "decision": "fs_read_permission_requested",
      "audit": {"permission": "CapFSRead", "granted": true, "user_confirmed": true}
    },
    {
      "id": "uuid-2",
      "timestamp": "2026-02-10T15:30:05Z",
      "message": {"type": "assistant_text", "content": "I'll read the README file for you."},
      "decision": "planning",
      "audit": {}
    },
    {
      "id": "uuid-3",
      "timestamp": "2026-02-10T15:30:06Z",
      "message": {"type": "assistant_action", "tool_name": "fs.read", "tool_args": {"path": "README.md"}},
      "decision": "tool_call_approved",
      "audit": {"tool": "fs.read", "permission_verified": true}
    },
    {
      "id": "uuid-4",
      "timestamp": "2026-02-10T15:30:07Z",
      "message": {"type": "tool_result", "tool_name": "fs.read", "success": true, "result": {...}},
      "decision": "tool_executed_successfully",
      "audit": {"execution_time_ms": 42, "result_size_bytes": 1024}
    }
  ]
}
```

---

## Acceptance Criteria

- [ ] All message types implemented with interfaces
- [ ] Each message type can serialize to API format and log format
- [ ] Conversation history tracks messages with IDs and timestamps
- [ ] Audit trail captures decision context for every turn
- [ ] LLM backend updated to use StreamStructured()
- [ ] Chat loop creates typed messages instead of generic Message
- [ ] All message serialization round-trips correctly
- [ ] Audit log can be exported to JSON for compliance/debugging

---

## Notes

- Message IDs enable attaching feedback, explanations, or "please reconsider" instructions
- Audit trail enables replay: could re-run same message sequence with different LLM
- Typed messages make it impossible to accidentally mix tool results with planning text
- Aligns with OpenAI Structured Outputs paradigm
