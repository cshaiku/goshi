# Action Plan: Streaming + Structured Output UX

**Status:** Not Started  
**Priority:** P1 (User experience critical)  
**Effort:** 1.5 days  
**Dependencies:** Issue #2 (Structured Messages), Issue #3 (Robust Parsing)  
**Blocks:** Phase 3

---

## Problem Statement

**Current Issue:** Streaming responsiveness conflicts with structured output validation
- LLM response streamed character-by-character for responsiveness
- But parser needs complete response to validate JSON schema
- User sees partial/incomplete text, then action happens invisibly
- No clear separation between planning phase and action phase
- Confusing UX: text appears, then tool runs, then final response without clear demarcation

**Risk:** Users don't understand what's happening; can't tell if tool was called or response is complete.

---

## Solution Design

### 1. Three-Phase Chat Flow

Establish clear phases with distinct UI treatment:

**Phase 1: Planning** (Streaming)
```
You: read the README

Goshi: [Planning...] I'll read the README file to get you the information...
```
User sees planning text stream, understands LLM is composing a response.

**Phase 2: Action** (Structured, Non-streaming)
```
[Taking action]
→ fs.read README.md
  Status: ✓ Complete (1024 bytes)
```
Single atomic block. User sees what tool is being called and knows it's happening.

**Phase 3: Report** (Streaming again, optional)
```
[Response from Goshi]
Here's the README content you requested:

# Project...
...
```
LLM can stream final response incorporating tool result. Or skip to next prompt.

### 2. Response Type Detection During Streaming

As response is received, buffer and detect type:

```go
type StreamBuffer struct {
    buf        strings.Builder
    jsonStart  int  // -1 if not found
    responseType ResponseType  // detected as we stream
}

func (sb *StreamBuffer) Write(chunk string) {
    before := sb.buf.Len()
    sb.buf.WriteString(chunk)
    after := sb.buf.Len()
    
    // If we haven't found JSON yet, keep looking
    if sb.jsonStart == -1 {
        if idx := strings.Index(sb.buf.String()[before:], "{"); idx != -1 {
            sb.jsonStart = before + idx
        }
    }
    
    // Once we have enough chars, try to detect type
    text := sb.buf.String()
    if sb.jsonStart != -1 && len(text) - sb.jsonStart > 50 {
        // Likely a JSON response; switch to buffering mode
    }
}

func (sb *StreamBuffer) Complete() ParsedResponse {
    return parser.Parse(sb.buf.String())
}
```

### 3. Smart Streaming Mode

```go
type StreamMode string

const (
    StreamModeText    StreamMode = "text"    // Stream all chars
    StreamModeBuffer  StreamMode = "buffer"  // Collect silently
    StreamModeStructured StreamMode = "structured"
)

func detectStreamMode(chunk string) StreamMode {
    if strings.Contains(chunk, `{"type":`) {
        return StreamModeStructured
    }
    return StreamModeText
}

// In chat loop:
var mode StreamMode = StreamModeText
buffer := &StreamBuffer{}

for {
    chunk, err := stream.Recv()
    if err != nil { break }
    
    // Auto-detect JSON early
    if mode == StreamModeText && strings.Contains(chunk, "{") {
        mode = StreamModeBuffer
        fmt.Print("\n[Taking action...")
    }
    
    switch mode {
    case StreamModeText:
        fmt.Print(chunk)  // Show immediately
        buffer.Write(chunk)
    case StreamModeBuffer:
        buffer.Write(chunk)  // Collect silently
    }
}

// Parse complete response
complete := buffer.Complete()
// ... handle by type
```

### 4. Clear Action UI

```go
type ActionPresentation struct {
    Tool    string
    Args    map[string]any
    Result  any
    Error   string
    DurationMs int
}

func (ap *ActionPresentation) Display() {
    fmt.Println("\n[Action]")
    
    // Show what we're doing
    fmt.Printf("→ %s", ap.Tool)
    for k, v := range ap.Args {
        fmt.Printf(" %s=%v", k, v)
    }
    fmt.Println()
    
    // Show result
    if ap.Error != "" {
        fmt.Printf("❌ Failed: %s\n", ap.Error)
    } else {
        fmt.Printf("✓ Complete (%dms)\n", ap.DurationMs)
        if bytes, ok := ap.Result.(map[string]any)["bytes"]; ok {
            fmt.Printf("  Size: %v\n", bytes)
        }
    }
    fmt.Println()
}
```

### 5. Session Display State

```go
type DisplayState struct {
    CurrentPhase DisplayPhase  // Planning / Acting / Reporting
    PlanningText strings.Builder
    CurrentAction *ActionPresentation
    ReportingText strings.Builder
}

type DisplayPhase string
const (
    PhasePlanning  DisplayPhase = "planning"
    PhaseActing    DisplayPhase = "acting"
    PhaseReporting DisplayPhase = "reporting"
)

// Allows rewriting status as we learn more
func (ds *DisplayState) UpdatePhase(phase DisplayPhase) {
    // Clear previous phase
    switch ds.CurrentPhase {
    case PhasePlanning:
        fmt.Println()  // Finish planning line
    case PhaseActing:
        ds.CurrentAction.Display()
    }
    
    ds.CurrentPhase = phase
    switch phase {
    case PhasePlanning:
        fmt.Print("\n[Planning] Goshi: ")
    case PhaseActing:
        // Covered by ActionPresentation
    case PhaseReporting:
        fmt.Print("\n[Response] Goshi: ")
    }
}
```

### 6. Full Chat Loop with Phases

```go
func runChat(systemPrompt string) {
    // ... initialization ...
    
    for {
        fmt.Print("You: ")
        line, _ := reader.ReadString('\n')
        line = strings.TrimSpace(line)
        
        messages.Add(&UserMessage{Content: line}, "user_input", nil)
        
        // Phase 1: PLANNING (streaming LLM response)
        fmt.Print("\n[Planning] Goshi: ")
        stream, _ := client.StreamStructured(ctx, system, messages.Messages())
        
        buffer := &StreamBuffer{}
        var streamMode StreamMode = StreamModeText
        
        for {
            chunk, err := stream.Recv()
            if err != nil { break }
            
            // Detect JSON early
            if streamMode == StreamModeText && detectStreamMode(chunk) == StreamModeStructured {
                streamMode = StreamModeBuffer
                fmt.Println("\n[Processing...]")
            }
            
            if streamMode == StreamModeText {
                fmt.Print(chunk)
                buffer.Write(chunk)
            } else {
                buffer.Write(chunk)
            }
        }
        stream.Close()
        
        parseResult := parser.ParseWithRetry(buffer.Complete())
        
        if !parseResult.Success {
            fmt.Printf("\n[Error] %s\n", parseResult.SuggestionMsg)
            messages.Add(
                &ErrorMessage{Code: "parse_error", Message: parseResult.SuggestionMsg},
                "parse_failed",
                map[string]any{"raw": buffer.buf.String()},
            )
            continue
        }
        
        // Phase 2: ACTION (if applicable)
        switch parseResult.Response.Type {
        case ResponseTypeText:
            fmt.Println()  // Line break after planning
            messages.Add(
                &AssistantTextMessage{Content: parseResult.Response.Text},
                "text_response",
                nil,
            )
            
        case ResponseTypeAction:
            call := parseResult.Response.Action
            actionMsg := &AssistantActionMessage{
                ToolName: call.Tool,
                ToolArgs: call.Args,
                ToolID:   uuid.New().String(),
            }
            messages.Add(actionMsg, "tool_call", nil)
            
            // Execute tool
            startTime := time.Now()
            result := router.Handle(app.ToolCall{Name: call.Tool, Args: call.Args})
            duration := time.Since(startTime)
            
            // Display action clearly
            presentation := &ActionPresentation{
                Tool:       call.Tool,
                Args:       call.Args,
                Result:     result,
                DurationMs: int(duration.Milliseconds()),
            }
            presentation.Display()
            
            // Add result to history
            messages.Add(
                &ToolResultMessage{
                    ToolID:     actionMsg.ToolID,
                    ToolName:   call.Tool,
                    Success:    result["error"] == nil,
                    Result:     result,
                    ExecutedAt: startTime,
                },
                "tool_executed",
                map[string]any{"duration_ms": presentation.DurationMs},
            )
            
            // Phase 3: REPORTING (ask LLM for follow-up)
            stream, _ = client.StreamStructured(ctx, system, messages.Messages())
            fmt.Print("[Response] Goshi: ")
            for {
                chunk, err := stream.Recv()
                if err != nil { break }
                fmt.Print(chunk)
            }
            fmt.Println("\n")
            stream.Close()
        }
        
        fmt.Println("-----------------------------------------------------")
    }
}
```

---

## Implementation Steps

1. **Create buffer types** (`cli/stream_buffer.go`)
   - StreamBuffer to accumulate response
   - Detect JSON early
   - Mode switching logic

2. **Create action display** (`cli/action_display.go`)
   - ActionPresentation struct
   - Display tool call, result, duration
   - Error formatting

3. **Refactor chat loop** (`cli/chat.go`)
   - Implement three-phase flow
   - Auto-detect JSON and switch modes
   - Display action atomically
   - Ask for LLM follow-up after tool

4. **Test UX paths** (`cli/chat_test.go`)
   - Text response (streaming only)
   - Text + tool response (planning + action + report)
   - Error handling with clear UI
   - Multiple tool calls in sequence

---

## UX Comparison

**Before:**
```
You: read README
Goshi: [streaming chars] I'll read the R E A D M E file for you...
[silently executes]
[back to prompt]
```

**After:**
```
You: read README

[Planning] Goshi: I'll read the README file for you...
[Processing...]
[Action]
→ fs.read README.md
✓ Complete (42ms)

[Response] Goshi: Here's the README content you requested:
# Project...
...

-----------------------------------------------------
```

---

## Acceptance Criteria

- [ ] Planning text streams character-by-character
- [ ] Action detection switches to buffer mode automatically
- [ ] Tool execution displays atomically (tool → result/error → duration)
- [ ] Follow-up response from LLM streams again
- [ ] Three phases (Planning / Acting / Reporting) clearly visually separated
- [ ] User can understand what happened at each step
- [ ] Test coverage for all three response types
- [ ] Error states display clearly with suggestions

---

## Notes

- Streaming provides responsiveness; buffering for JSON provides clarity
- Auto-detection means no explicit mode switch needed; happens naturally
- Action display must be atomic to avoid confusion about what happened
- Three-phase flow matches human mental model of: "What will you do?" → "Here's what I did" → "Here's what it means"
