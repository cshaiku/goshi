# Action Plan: Unified Tool Registry System

**Status:** Not Started  
**Priority:** P0 (Blocker for all phases)  
**Effort:** 2 days  
**Dependencies:** None  
**Blocks:** Phase 2, 3, 4

---

## Problem Statement

**Current Issue:** Dual tool detection paths create inconsistency and authorization gaps
- Regex-based pre-detection in `chat.go` (lines 127-145) detects fs.read/fs.write and auto-grants
- Separate tool call parsing in `chat_tools.go` attempts to handle JSON tool calls
- Inconsistent permission enforcement: one path checks dynamically, one checks at tool routing
- New tools require changes in multiple places (regex, tool router, dispatcher)
- No centralized registry of what tools exist, their schemas, permissions, or constraints

**Risk:** Tool calls can be silently ignored, permissions bypassed, or new tools added without full integration.

---

## Solution Design

### 1. ToolRegistry Data Structure

Create `internal/app/tool_registry.go`:

```go
type ToolDefinition struct {
    ID              string                 // "fs.read", "fs.write", "fs.list"
    Name            string                 // Human-readable name
    Description     string                 // What the tool does
    RequiredPermission Capability           // CapFSRead, CapFSWrite
    Schema          JSONSchema             // Input validation schema
    MaxRetries      int
}

type ToolRegistry struct {
    tools map[string]ToolDefinition
    mu    sync.RWMutex
}

// Core methods:
// - Register(definition) error          // Add or update tool
// - Get(id) (ToolDefinition, bool)      // Lookup tool
// - All() []ToolDefinition              // List all tools
// - ValidateCall(id, args) error        // Validate against schema
```

### 2. Tool Registration

Define all tools at init time in `internal/app/tools.go`:

```go
var (
    FSReadTool = ToolDefinition{
        ID: "fs.read",
        Name: "Read File",
        Description: "Read contents of a file from the repository",
        RequiredPermission: CapFSRead,
        Schema: JSONSchema{
            Type: "object",
            Properties: map[string]JSONSchema{
                "path": {Type: "string", Description: "Relative path to file"},
            },
            Required: []string{"path"},
        },
    }
    // ... other tools
)

func NewToolRegistry() *ToolRegistry {
    r := &ToolRegistry{tools: make(map[string]ToolDefinition)}
    r.Register(FSReadTool)
    r.Register(FSWriteTool)
    r.Register(FSListTool)
    return r
}
```

### 3. Tool Router Integration

Update `tool_router.go` to use registry:

```go
func (r *ToolRouter) Handle(call ToolCall) Response {
    // Look up tool definition
    toolDef, ok := r.registry.Get(call.Name)
    if !ok {
        return ErrorResponse("unknown tool")
    }
    
    // Validate call against schema
    if err := r.registry.ValidateCall(call.Name, call.Args); err != nil {
        return ErrorResponse("invalid tool call: " + err.Error())
    }
    
    // Check capability
    if !r.caps.Has(toolDef.RequiredPermission) {
        return ErrorResponse("permission denied for tool: " + toolDef.ID)
    }
    
    // Execute
    return r.dispatcher.Dispatch(call.Name, call.Args)
}
```

### 4. Remove Regex Detection

Delete the entire pre-detection block in `chat.go`. The LLM will handle all tool invocations through the structured output.

---

## Implementation Steps

1. **Create types** (`tool_registry.go`)
   - ToolDefinition struct with all metadata
   - ToolRegistry with CRUD operations
   - JSONSchema validation helper

2. **Define tools** (`tools.go`)
   - Register fs.read, fs.write, fs.list with complete schemas
   - Document each tool's purpose and constraints
   - Make extensible for future tools

3. **Update tool router** (`tool_router.go`)
   - Change constructor to accept registry
   - Update Handle() to use registry lookups
   - Change error messages to use tool metadata

4. **Update chat loop** (`chat.go`)
   - Remove lines 127-165 (regex detection)
   - Pass registry to router
   - Simplify permission grant logic (now tool-driven)

5. **Tests** (`tool_registry_test.go`, `tool_router_test.go`)
   - Test registry CRUD operations
   - Test schema validation for each tool
   - Test unknown tool handling
   - Test permission enforcement through registry

---

## JSON Schema Example

```json
{
  "fs.read": {
    "type": "object",
    "description": "Read a file from the repository",
    "properties": {
      "path": {
        "type": "string",
        "description": "Relative path within repository",
        "pattern": "^[^/].*$"
      }
    },
    "required": ["path"],
    "additionalProperties": false
  }
}
```

---

## Acceptance Criteria

- [ ] ToolRegistry compiles and passes unit tests
- [ ] All three tools (fs.read, fs.write, fs.list) registered with complete schemas
- [ ] Tool router uses registry for all lookups
- [ ] No regex-based tool detection in chat.go
- [ ] Schema validation rejects invalid tool calls
- [ ] Permission enforcement still works through registry
- [ ] Tool metadata visible for help/discovery (future feature)

---

## Notes

- This is the foundational piece; all subsequent phases depend on it
- Schemas will be passed to LLM in system prompt for precise tool calling
- Registry makes it trivial to add new tools: just create ToolDefinition and Register
- Can extend to include retry policies, rate limits, audit flags per tool
