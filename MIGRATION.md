# Migration Guide: LLM Integration v1

This guide explains how to migrate from previous goshi integrations to the new structured LLM integration.

## Overview

The new LLM integration introduces:
- **Structured Message Types** — Type-safe communication protocol
- **Tool Registry** — Dynamic tool discovery with schema validation
- **Permission Model** — Fine-grained capability-based access control
- **Audit Trail** — Complete record of all security-relevant events
- **Chat Sessions** — Encapsulated conversation state

## Key Changes

### 1. Message Types

**Old Approach:**
```go
type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}
```

**New Approach:**
```go
// Type-safe interfaces
type LLMMessage interface{}

type UserMessage struct {
    Content string
}

type AssistantTextMessage struct {
    Content string
}

type AssistantActionMessage struct {
    ToolName string
    ToolArgs map[string]any
    ToolID   string
}

type ToolResultMessage struct {
    ToolName string
    Result   interface{}
}
```

**Migration:**
Use the new types when building message history:
```go
session.AddUserMessage("your message")
session.AddAssistantTextMessage("response")
session.AddAssistantActionMessage("tool.name", map[string]any{})
session.AddToolResultMessage("tool.name", result)
```

### 2. Tool Registry

**Old Approach:**
Tools were hard-coded or loosely defined.

**New Approach:**
Tools are automatically discovered from the action dispatcher with full schema validation:

```go
// Get available tools
toolDefs := session.ToolRouter.GetToolDefinitions()

// Each tool has:
// - ID: "fs.read", "fs.write", etc.
// - Description: Human-readable description
// - Schema: JSON Schema for argument validation

for _, tool := range toolDefs {
    fmt.Printf("Tool: %s\n", tool.ID)
    fmt.Printf("Description: %s\n", tool.Description)
    // Schema is validated automatically
}
```

**Benefits:**
- No manual tool specification needed
- Schema validation prevents invalid calls
- Self-documenting tool interface

### 3. Permission Model

**Old Approach:**
No explicit permission model.

**New Approach:**
Capabilities are explicitly granted and tracked:

```go
// Initially, no permissions
if session.HasPermission("FS_READ") {
    // Won't execute
}

// Grant permission
session.GrantPermission("FS_READ")

// Now it works
result := session.ToolRouter.Handle(app.ToolCall{
    Name: "fs.read",
    Args: map[string]any{"path": "file.txt"},
})

// Denied attempts are logged
session.DenyPermission("FS_WRITE")
// Next FS_WRITE call will fail with permission error
```

**Permission Types:**
- `FS_READ` — Read files and directories
- `FS_WRITE` — Write and modify files

**Audit Record:**
Each permission decision is logged:
```go
auditLog := session.GetAuditLog()
// Output:
// [2026-02-10 12:00:00] GRANT FS_READ at /Users/cs/goshi
// [2026-02-10 12:00:05] FS_READ ALLOW fs.read path=file.txt
```

### 4. Chat Session Lifecycle

**Old Approach:**
Manual message management and state tracking.

**New Approach:**
Encapsulated session management:

```go
// Create session
session, err := cli.NewChatSession(
    ctx,
    "You are a helpful assistant.",
    backend, // LLM backend
)
if err != nil {
    return err
}

// Session automatically manages:
// - Message history (with types)
// - Permission state
// - Working directory
// - Tool routing

// Add messages
session.AddUserMessage("What files are here?")

// Execute tools (permission checked internally)
result := session.ToolRouter.Handle(app.ToolCall{
    Name: "fs.list",
    Args: map[string]any{"path": "."},
})

// Convert to legacy format if needed (backward compatibility)
legacyMessages := session.ConvertMessagesToLegacy()
```

### 5. Tool Execution

**Old Approach:**
Direct tool calls without validation.

**New Approach:**
Strict validation pipeline:

```go
// All these are checked automatically:
// 1. Tool exists in registry
// 2. Capability is granted
// 3. Arguments match schema
// 4. Tool is executed
// 5. Result is returned

result := session.ToolRouter.Handle(app.ToolCall{
    Name: "fs.read",
    Args: map[string]any{"path": "file.txt"},
})

// Result is either:
// - Success: {"result": "file contents"}
// - Permission Error: {"error": "permission denied: FS_READ"}
// - Validation Error: {"error": "validation failed: path is required"}
// - Unknown Tool: {"error": "unknown tool: invalid.tool"}
```

### 6. LLM Backend Integration

**Old Approach:**
Direct stream handling.

**New Approach:**
Backend interface is cleaner and more testable:

```go
type Backend interface {
    Stream(ctx context.Context, system string, messages []Message) (Stream, error)
}

// Usage in session
session, _ := cli.NewChatSession(ctx, systemPrompt, backend)

// Backends can be:
// - Ollama (local)
// - Mock (testing)
// - Custom (your own implementation)
```

## Migration Steps

### Step 1: Update Message Construction
Replace loose string-based messages with typed messages:

```go
// Old
messages := []Message{
    {Role: "user", Content: "Hello"},
}

// New
session.AddUserMessage("Hello")
// Messages are now properly typed internally
```

### Step 2: Update Tool Definitions
Let the tool registry auto-discover tools instead of manual definition:

```go
// Old - Manual tool listing
tools := []ToolDef{}

// New - Auto-discovered
toolDefs := session.ToolRouter.GetToolDefinitions()
// All tools with schemas are available
```

### Step 3: Add Permission Grants
Explicitly grant capabilities before tool use:

```go
// Before using tools that read files
session.GrantPermission("FS_READ")

// Before using tools that write files
session.GrantPermission("FS_WRITE")
```

### Step 4: Update Tool Calls
Use the new tool router pattern:

```go
// Old
result := executeToolDirectly(toolName, args)

// New
result := session.ToolRouter.Handle(app.ToolCall{
    Name: toolName,
    Args: args,
})
```

### Step 5: Track Audit Trail
Use the audit log for compliance and debugging:

```go
// Monitor all permission decisions
log.Info(session.GetAuditLog())

// Results include timestamps and context
```

## Breaking Changes

| Item | Old | New | Action |
|------|-----|-----|--------|
| Messages | `Message{Role, Content}` | Typed interfaces | Use session methods |
| Tool Registry | Manual | Auto-discovered | Remove manual specs |
| Permissions | None | Explicit grants | Add permission checks |
| Tool Execution | Direct | Via router | Use `session.ToolRouter.Handle()` |
| Message Types | Untyped | Strongly typed | Use new types |
| Audit Trail | Not available | Comprehensive | Leverage for logging |

## Backward Compatibility

The new integration maintains backward compatibility through:

1. **Message Conversion** — Convert structured messages to legacy format:
   ```go
   legacyMessages := session.ConvertMessagesToLegacy()
   ```

2. **Tool Result Format** — Results are compatible with existing code expecting error/result maps

3. **Session API** — Can be wrapped to provide legacy interface if needed

## Testing Your Migration

Use the new integration tests as reference:

```bash
# Run integration tests
go test -run Integration ./internal/cli -v

# Run specific test category
go test -run Integration_Permission ./internal/cli -v

# Run all CLI tests
go test ./internal/cli -v
```

Key test scenarios to verify:
- ✅ Tool execution with proper permissions
- ✅ Permission denial errors
- ✅ Schema validation errors
- ✅ Message history tracking
- ✅ Audit trail logging
- ✅ Full 6-phase chat flow

## Common Issues

### Issue: Tool Returns Permission Error

**Cause:** Capability not granted.

**Solution:**
```go
if !session.HasPermission("FS_READ") {
    session.GrantPermission("FS_READ")
}
```

### Issue: Tool Returns Validation Error

**Cause:** Arguments don't match schema.

**Solution:**
```go
// Check what schema expects
toolDefs := session.ToolRouter.GetToolDefinitions()
// Look at tool schema for required fields
// All required arguments must be in Args map
```

### Issue: Unknown Tool Error

**Cause:** Tool name doesn't match registry.

**Solution:**
```go
// List available tools
toolDefs := session.ToolRouter.GetToolDefinitions()
for _, tool := range toolDefs {
    fmt.Println(tool.ID) // Use these IDs
}
```

## Questions?

See [README.md](README.md) for LLM Integration Architecture details or review integration tests at [internal/cli/chat_integration_test.go](internal/cli/chat_integration_test.go).
