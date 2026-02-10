# Action Plan: Robust Structured Response Parsing

**Status:** Not Started  
**Priority:** P0 (Security-critical)  
**Effort:** 1 day  
**Dependencies:** Issue #1 (Tool Registry), Issue #2 (Structured Messages)  
**Blocks:** Phase 3, 4

---

## Problem Statement

**Current Issue:** Fragile tool parsing creates security and reliability gaps
- Uses `strings.Index("{")` and `strings.LastIndex("}")` to find JSON (chat_tools.go, lines 10-11)
- Vulnerable to tool calls embedded in conversational filler
- Minimal JSON validation; silently fails on malformed JSON
- Cannot distinguish between: partial responses, multiple calls, escaped JSON in text
- No schema validation of tool arguments; relies on downstream dispatcher
- Parse failures are silent (returns `nil, false`) making debugging difficult

**Risk:** LLM could lose tool calls in noise, call with invalid args, or accidentally invoke tools embedded in examples.

---

## Solution Design

### 1. Structured Output Format

Establish clear contract: LLM outputs EITHER:

**Option A: Pure Text** (streaming)
```
I'll help you read that file. Let me fetch it for you.
```

**Option B: Single Tool Call** (after any text)
```json
{
  "type": "action",
  "tool": "fs.read",
  "args": {
    "path": "README.md"
  }
}
```

**Option C: Error** (when LLM wants to report constraint)
```json
{
  "type": "error",
  "code": "permission_denied|invalid_args|tool_not_found",
  "message": "I don't have permission to read that file"
}
```

LLM instructions make this explicit and required.

### 2. Parser with Schema Validation

Create `internal/llm/response_parser.go`:

```go
type ResponseType string

const (
    ResponseTypeText   ResponseType = "text"
    ResponseTypeAction ResponseType = "action"
    ResponseTypeError  ResponseType = "error"
)

type ParsedResponse struct {
    Type       ResponseType
    
    // For text
    Text       string
    
    // For action
    Action     *ActionCall
    
    // For error
    ErrorCode  string
    ErrorMsg   string
    
    // Metadata
    RawText    string
    ParseError string
}

type ActionCall struct {
    Tool string
    Args map[string]any
}

// Parser validates against tool registry and JSON schemas
type ResponseParser struct {
    registry *ToolRegistry
}

func (p *ResponseParser) Parse(text string) ParsedResponse {
    result := ParsedResponse{RawText: text}
    
    // Step 1: Try to extract JSON (if any)
    jsonStart := strings.Index(text, "{")
    jsonEnd := strings.LastIndex(text, "}")
    
    if jsonStart == -1 || jsonEnd == -1 {
        // No JSON, treat as text
        result.Type = ResponseTypeText
        result.Text = text
        return result
    }
    
    // Step 2: Validate JSON structure
    jsonStr := text[jsonStart : jsonEnd+1]
    var payload struct {
        Type  string         `json:"type"`
        Tool  string         `json:"tool"`
        Args  map[string]any `json:"args"`
        Code  string         `json:"code"`
        Message string       `json:"message"`
    }
    
    if err := json.Unmarshal([]byte(jsonStr), &payload); err != nil {
        result.ParseError = fmt.Sprintf("invalid JSON: %v", err)
        // Treat as text if JSON is malformed
        result.Type = ResponseTypeText
        result.Text = strings.TrimSpace(text[:jsonStart] + text[jsonEnd+1:])
        return result
    }
    
    // Step 3: Validate type discriminator
    switch payload.Type {
    case "action":
        return p.parseAction(payload, result)
    case "error":
        return p.parseError(payload, result)
    default:
        result.Type = ResponseTypeText
        result.Text = text
        result.ParseError = fmt.Sprintf("unknown response type: %s", payload.Type)
        return result
    }
}

func (p *ResponseParser) parseAction(payload struct{...}, result ParsedResponse) ParsedResponse {
    // Step 4a: Tool must exist
    toolDef, ok := p.registry.Get(payload.Tool)
    if !ok {
        result.Type = ResponseTypeError
        result.ErrorCode = "tool_not_found"
        result.ErrorMsg = fmt.Sprintf("unknown tool: %s", payload.Tool)
        return result
    }
    
    // Step 4b: Args must match schema
    if err := p.registry.ValidateCall(payload.Tool, payload.Args); err != nil {
        result.Type = ResponseTypeError
        result.ErrorCode = "invalid_args"
        result.ErrorMsg = fmt.Sprintf("invalid args for %s: %v", payload.Tool, err)
        return result
    }
    
    result.Type = ResponseTypeAction
    result.Action = &ActionCall{
        Tool: payload.Tool,
        Args: payload.Args,
    }
    return result
}

func (p *ResponseParser) parseError(payload struct{...}, result ParsedResponse) ParsedResponse {
    result.Type = ResponseTypeError
    result.ErrorCode = payload.Code
    result.ErrorMsg = payload.Message
    return result
}
```

### 3. Enhanced Error Reporting

```go
type ParseResult struct {
    Success        bool
    Response       ParsedResponse
    ValidationErr  error
    SuggestionMsg string
}

func (p *ResponseParser) ParseWithRetry(text string) ParseResult {
    parsed := p.Parse(text)
    
    if parsed.Type == ResponseTypeError && parsed.ErrorCode != "" {
        // LLM reported an error; don't retry
        return ParseResult{
            Success:  false,
            Response: parsed,
            ValidationErr: errors.New(parsed.ErrorMsg),
        }
    }
    
    if parsed.ParseError != "" {
        return ParseResult{
            Success:       false,
            Response:      parsed,
            ValidationErr: errors.New(parsed.ParseError),
            SuggestionMsg: "The response couldn't be parsed. Ask the LLM to clarify or retry.",
        }
    }
    
    return ParseResult{
        Success:  true,
        Response: parsed,
    }
}
```

### 4. JSON Schema Enforcement in LLM Prompt

Update ollama/client.go toolInstructions:

```go
const toolInstructions = `
# TOOL USE - STRUCTURED OUTPUT REQUIRED

When you want to read, list, or write files, respond with ONLY a JSON object (no other text).

## Available Tools

### fs.read
{
  "type": "action",
  "tool": "fs.read",
  "args": {
    "path": "relative/path/to/file.txt"
  }
}

### fs.write
{
  "type": "action",
  "tool": "fs.write",
  "args": {
    "path": "relative/path/to/file.txt",
    "content": "file contents here"
  }
}

### fs.list
{
  "type": "action",
  "tool": "fs.list",
  "args": {
    "path": "."
  }
}

## Error Cases

If you cannot proceed, respond with:
{
  "type": "error",
  "code": "permission_denied|invalid_args|tool_not_found",
  "message": "explanation"
}

## Important
- Respond with ONLY the JSON object when making a tool call
- Include "type": "action" always
- All paths must be relative and within the repository
- Do not invent file contents; use tools to read them
`
```

### 5. Chat Loop Integration

Update chat.go to use parser:

```go
parser := llm.NewResponseParser(toolRegistry)

// In main loop, after LLM stream:
fmt.Print("Goshi: ")
var reply strings.Builder
for {
    chunk, err := stream.Recv()
    if err != nil { break }
    fmt.Print(chunk)
    reply.WriteString(chunk)
}
fmt.Println()

// Parse response
parseResult := parser.ParseWithRetry(reply.String())
if !parseResult.Success {
    fmt.Printf("Error: %s\n", parseResult.SuggestionMsg)
    // Can ask LLM to retry or ask user for next input
    continue
}

// Handle response
switch parseResult.Response.Type {
case ResponseTypeText:
    messages = append(messages, &AssistantTextMessage{Content: parseResult.Response.Text})
    
case ResponseTypeAction:
    call := parseResult.Response.Action
    messages = append(messages, &AssistantActionMessage{ToolName: call.Tool, ToolArgs: call.Args})
    
    result := router.Handle(app.ToolCall{Name: call.Tool, Args: call.Args})
    messages = append(messages, &ToolResultMessage{/*...*/})
    
    // Ask LLM for follow-up
    stream, _ = client.StreamStructured(ctx, system, messages)
    // ... handle follow-up
    
case ResponseTypeError:
    fmt.Printf("LLM Error: %s\n", parseResult.Response.ErrorMsg)
    messages = append(messages, &ErrorMessage{Code: parseResult.Response.ErrorCode, Msg: parseResult.Response.ErrorMsg})
}
```

---

## Implementation Steps

1. **Define response types** (`llm/response_parser.go`)
   - ResponseType enum and ParsedResponse struct
   - Document expected JSON formats

2. **Implement parser** (`llm/response_parser.go`)
   - Parse() method with JSON extraction
   - Type discrimination (action/error/text)
   - Schema validation delegation to registry

3. **Add error reporting** (`llm/response_parser.go`)
   - ParseWithRetry() with helpful messages
   - Distinguish: validation errors vs parse errors

4. **Update LLM prompt** (`ollama/client.go`)
   - Add JSON schema examples for each tool
   - Make structured output mandatory
   - Clarify error response format

5. **Integrate with chat loop** (`cli/chat.go`)
   - Use parser instead of TryHandleToolCall()
   - Handle all response types (text, action, error)
   - Add retry logic for parse failures

6. **Tests** (`llm/response_parser_test.go`)
   - Valid tool calls parse correctly
   - Invalid JSON treated as text
   - Schema validation rejects bad args
   - Error responses parsed correctly
   - Ambiguous cases (multiple JSON blocks) handled safely

---

## Security Properties Enforced

1. ✅ Tool calls must be explicit JSON with type="action"
2. ✅ All tool arguments validated against registry schema
3. ✅ Unknown tools rejected before execution
4. ✅ Parse errors don't silently fail
5. ✅ LLM cannot invoke tools via embedded examples
6. ✅ All validation happens before dispatcher.Dispatch()

---

## Acceptance Criteria

- [ ] Parser handles text, action, error response types correctly
- [ ] JSON schema validation rejects invalid tool arguments
- [ ] Unknown tools produce clear error messages
- [ ] Parse failures are logged with suggestions
- [ ] LLM prompt specifies structured output requirements
- [ ] Chat loop uses parser for all LLM responses
- [ ] 100% of test cases pass validation
- [ ] No tool calls can be silently ignored

---

## Notes

- Parser delegates tool lookup and schema validation to registry (DRY principle)
- Fragile substring-based parsing replaced with explicit type discrimination
- Error responses from LLM are first-class; don't treat as parse failures
- Can extend to support multiple sequential tool calls in future
