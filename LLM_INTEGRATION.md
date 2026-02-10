# LLM Integration Reference

Quick reference guide for using goshi's LLM integration with tool calling and permissions.

## Quick Start

### 1. Create a Chat Session

```go
package main

import (
    "context"
    "github.com/cshaiku/goshi/internal/cli"
    "github.com/cshaiku/goshi/internal/llm"
)

func main() {
    ctx := context.Background()
    
    // Use real backend (Ollama, etc.) or MockLLMBackend for testing
    backend := createLLMBackend()
    
    // Create session with system prompt
    session, err := cli.NewChatSession(ctx, 
        "You are a helpful filesystem assistant.", 
        backend,
    )
    if err != nil {
        panic(err)
    }
    
    // Session ready to use
    _ = session
}
```

### 2. Grant Permissions

```go
// Check current permissions
if !session.HasPermission("FS_READ") {
    session.GrantPermission("FS_READ")
}

// Grant write capability
session.GrantPermission("FS_WRITE")

// View audit trail
auditLog := session.GetAuditLog()
println(auditLog)
```

### 3. Add User Input

```go
session.AddUserMessage("List all files in the current directory")
```

### 4. Parse LLM Response

```go
// LLM returns a response that looks like:
// {"type": "action", "action": {"tool": "fs.list", "args": {"path": "."}}}

response := callLLM(session)
structured, err := llm.ParseStructuredResponse(response)
if err != nil {
    panic(err)
}

// Extract tool call
if structured.Type == llm.ResponseTypeAction {
    toolName := structured.Action.Tool
    args := structured.Action.Args
    
    // Execute through permission-checked router
    result := session.ToolRouter.Handle(app.ToolCall{
        Name: toolName,
        Args: args,
    })
    
    // Add to history
    session.AddAssistantActionMessage(toolName, args)
    session.AddToolResultMessage(toolName, result)
}
```

## Message Types

### UserMessage
User input to the chat.

```go
session.AddUserMessage("What files are in this directory?")
```

### AssistantTextMessage
LLM's text response.

```go
session.AddAssistantTextMessage("Here are the files in the directory...")
```

### AssistantActionMessage
LLM requesting a tool call.

```go
session.AddAssistantActionMessage("fs.list", map[string]any{
    "path": ".",
})
```

### ToolResultMessage
Result from executing a tool.

```go
result := session.ToolRouter.Handle(app.ToolCall{
    Name: "fs.list",
    Args: map[string]any{"path": "."},
})

session.AddToolResultMessage("fs.list", result)
```

## Available Tools

### fs.list
List files in a directory (requires `FS_READ`).

```go
result := session.ToolRouter.Handle(app.ToolCall{
    Name: "fs.list",
    Args: map[string]any{"path": "."},
})
```

### fs.read
Read file contents (requires `FS_READ`).

```go
result := session.ToolRouter.Handle(app.ToolCall{
    Name: "fs.read",
    Args: map[string]any{"path": "file.txt"},
})
```

### fs.write
Write or modify a file (requires `FS_WRITE`).

```go
result := session.ToolRouter.Handle(app.ToolCall{
    Name: "fs.write",
    Args: map[string]any{
        "path":    "output.txt",
        "content": "Hello World",
    },
})
```

## Tool Execution Patterns

### Happy Path (Tool Succeeds)

```go
// 1. Grant permission
session.GrantPermission("FS_READ")

// 2. Call tool
result := session.ToolRouter.Handle(app.ToolCall{
    Name: "fs.read",
    Args: map[string]any{"path": "file.txt"},
})

// 3. Result is: {"result": "file contents..."}
if resultMap, ok := result.(map[string]any); ok {
    if content, hasResult := resultMap["result"]; hasResult {
        println(content.(string))
    }
}
```

### Permission Denied

```go
// Don't grant permission
// session.GrantPermission("FS_READ") // <- not called

result := session.ToolRouter.Handle(app.ToolCall{
    Name: "fs.read",
    Args: map[string]any{"path": "file.txt"},
})

// Result is: {"error": "permission denied: FS_READ"}
if resultMap, ok := result.(map[string]any); ok {
    if err, hasError := resultMap["error"]; hasError {
        println("Error:", err.(string))
    }
}
```

### Validation Error

```go
session.GrantPermission("FS_READ")

// Missing required "path" argument
result := session.ToolRouter.Handle(app.ToolCall{
    Name: "fs.read",
    Args: map[string]any{}, // Empty!
})

// Result is: {"error": "validation failed: path is required"}
```

### Wrong Argument Type

```go
session.GrantPermission("FS_READ")

// "path" should be string, not number
result := session.ToolRouter.Handle(app.ToolCall{
    Name: "fs.read",
    Args: map[string]any{"path": 12345}, // Wrong type!
})

// Result is: {"error": "validation failed: ..."}
```

## Structured Response Examples

### Text Response
```json
{
  "type": "text",
  "text": "Here's what I found..."
}
```

```go
sr, _ := llm.ParseStructuredResponse(jsonStr)
if sr.Type == llm.ResponseTypeText {
    message := sr.Text
    session.AddAssistantTextMessage(message)
}
```

### Action Response
```json
{
  "type": "action",
  "action": {
    "tool": "fs.list",
    "args": {"path": "."}
  }
}
```

```go
sr, _ := llm.ParseStructuredResponse(jsonStr)
if sr.Type == llm.ResponseTypeAction {
    toolName := sr.Action.Tool
    args := sr.Action.Args
    result := session.ToolRouter.Handle(app.ToolCall{
        Name: toolName,
        Args: args,
    })
    session.AddToolResultMessage(toolName, result)
}
```

### Malformed JSON (Falls Back to Text)
```go
sr, _ := llm.ParseStructuredResponse("{invalid json}")
if sr.Type == llm.ResponseTypeText {
    // Falls back to treating as plain text
    session.AddAssistantTextMessage(sr.Text)
}
```

## Permissions Reference

### Available Capabilities

| Capability | Tools Enabled | Use Case |
|------------|---------------|----------|
| `FS_READ` | `fs.list`, `fs.read` | Reading files and directories |
| `FS_WRITE` | `fs.write` | Modifying or creating files |

### Permission Commands

```go
// Grant capability
session.GrantPermission("FS_READ")

// Check if granted
if session.HasPermission("FS_READ") {
    // Tool calls with FS_READ will succeed
}

// Deny capability
session.DenyPermission("FS_WRITE")

// View all permission events
auditLog := session.GetAuditLog()
// Output format:
// [2026-02-10 12:00:00] GRANT FS_READ at /project/dir
// [2026-02-10 12:00:05] FS_READ ALLOW fs.read path=file.txt
```

## Tool Registry Discovery

```go
// Get all available tools
toolDefs := session.ToolRouter.GetToolDefinitions()

for _, toolDef := range toolDefs {
    fmt.Printf("Tool: %s\n", toolDef.ID)
    fmt.Printf("Description: %s\n", toolDef.Description)
    
    // Schema validation info
    schema := toolDef.Schema
    println("Schema type:", schema.Type)
    println("Required fields:", schema.Required)
}

// Output:
// Tool: fs.list
// Description: List files in a directory
// Schema type: object
// Required fields: [path]
//
// Tool: fs.read
// Description: Read file contents
// Schema type: object
// Required fields: [path]
//
// Tool: fs.write
// Description: Write or create a file
// Schema type: object
// Required fields: [path content]
```

## Full Chat Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/cshaiku/goshi/internal/app"
    "github.com/cshaiku/goshi/internal/cli"
    "github.com/cshaiku/goshi/internal/llm"
)

func main() {
    ctx := context.Background()
    backend := createLLMBackend() // Create your backend
    
    // Create session
    session, _ := cli.NewChatSession(ctx, "You are a helpful assistant.", backend)
    
    // Grant permissions for this conversation
    session.GrantPermission("FS_READ")
    session.GrantPermission("FS_WRITE")
    
    // Phase 1: Listen
    session.AddUserMessage("List the files and read the first one")
    
    // Phase 2-3: Detect & Plan (LLM decides)
    llmResponse := callLLMWithHistory(session)
    
    // Phase 4: Parse
    structured, _ := llm.ParseStructuredResponse(llmResponse)
    
    // Phase 5: Act
    if structured.Type == llm.ResponseTypeAction {
        result := session.ToolRouter.Handle(app.ToolCall{
            Name: structured.Action.Tool,
            Args: structured.Action.Args,
        })
        
        // Phase 6: Report
        session.AddAssistantActionMessage(structured.Action.Tool, structured.Action.Args)
        session.AddToolResultMessage(structured.Action.Tool, result)
        
        fmt.Printf("Tool executed: %s\n", structured.Action.Tool)
        fmt.Printf("Result: %v\n", result)
    }
    
    // Access audit trail
    fmt.Println("\nAudit Trail:")
    fmt.Println(session.GetAuditLog())
}
```

## Testing

Use MockLLMBackend for deterministic testing:

```go
import "github.com/cshaiku/goshi/internal/cli"

func TestMyIntegration(t *testing.T) {
    // Create mock backend with scripted responses
    response := `{"type": "action", "action": {"tool": "fs.list", "args": {"path": "."}}}`
    backend := cli.NewMockLLMBackend(t, response)
    
    // Create session
    session, _ := cli.NewChatSession(context.Background(), "Test", backend)
    session.GrantPermission("FS_READ")
    
    // Execute
    session.AddUserMessage("List files")
    result := session.ToolRouter.Handle(app.ToolCall{
        Name: "fs.list",
        Args: map[string]any{"path": "."},
    })
    
    // Verify
    if resultMap, ok := result.(map[string]any); ok {
        if _, hasError := resultMap["error"]; hasError {
            t.Fatal("Unexpected error")
        }
    }
}
```

## See Also

- [README.md](README.md) — Architecture overview
- [MIGRATION.md](MIGRATION.md) — Upgrade guide
- [internal/cli/chat_integration_test.go](internal/cli/chat_integration_test.go) — Integration test examples
