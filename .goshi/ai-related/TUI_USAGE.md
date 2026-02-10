# Goshi TUI (Text User Interface) Usage Guide

## Overview

Goshi now features a modern Text User Interface (TUI) built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), providing an interactive chat experience directly in your terminal.

## Modes

### TUI Mode (Default)

The TUI provides an interactive, visual chat interface with:
- **Scrollable message history** - Review past conversations
- **Syntax-highlighted messages** - Color-coded user/assistant messages
- **Live status display** - See enforcement status and law metrics
- **Keyboard shortcuts** - Efficient navigation and control

**Launch:**
```bash
goshi
```

### Headless/CLI Mode

For automation, scripts, and pipelines, use headless mode:
```bash
goshi --headless
```

This provides traditional stdio-based interaction suitable for:
- Shell scripts
- CI/CD pipelines
- Automated testing
- Output piping and redirection

## TUI Features

### Interface Layout

```
╔═ GOSHI TUI ════════════════════════════════════════════╗
║ Laws: X lines │ Constraints: Y │ Status: ACTIVE/STAGED
╚═══════════════════════════════════════════════════════╝

┌─ Chat History (scrollable) ─────────────────────────┐
│                                                      │
│ You: Hello, can you help me?                        │
│                                                      │
│ Goshi: Of course! How can I assist you today?       │
│                                                      │
└──────────────────────────────────────────────────────┘

─ Ready ──────────────────────────────────────────────

┌─ Input (Ctrl+S to send)
│ Type your message here...
│
│
```

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+S` | Send the current message |
| `Ctrl+C` or `Esc` | Quit the TUI |
| `↑` / `↓` | Scroll through chat history |
| `PgUp` / `PgDn` | Page up/down in history |

### Status Indicators

**Header Status:**
- `ENFORCEMENT STAGED` - No permissions granted yet
- `ENFORCEMENT ACTIVE (FS_READ)` - Read permission granted
- `ENFORCEMENT ACTIVE (FS_WRITE)` - Write permission granted
- `ENFORCEMENT ACTIVE (FS_READ + FS_WRITE)` - Both permissions granted

**Status Line:**
- `Ready` - Waiting for input
- `Thinking...` - LLM is processing (future: Phase 5)
- `Error: ...` - Error messages displayed in red

### Message Styling

- **Your messages** appear in orange/amber
- **Goshi's responses** appear in mint green
- **Welcome text** appears in muted gray
- **Error messages** appear in red

## Architecture

The TUI is built on a clean Model-Update-View (Elm Architecture) pattern:

```
internal/tui/
├── tui.go           - Main TUI logic, model, and integration
└── (future)         - Additional components as needed
```

**Key Components:**
- `model` - Application state (messages, viewport, textarea)
- `Update()` - Handles events (keypress, window resize)
- `View()` - Renders the UI
- `Run()` - Initializes and starts the TUI program

## Integration with Chat Session

The TUI integrates with Goshi's existing chat session management and LLM backend:

```go
// TUI mode initialization
session, _ := session.NewChatSession(ctx, systemPrompt, backend)
tui.Run(systemPrompt, session)
```

**LLM Streaming Flow:**
1. User sends message → `handleSendMessage()` called
2. Message added to session history
3. Async `streamLLMResponse()` command started
4. Backend streams response chunks
5. `llmCompleteMsg` sent when streaming finishes
6. Response parsed and routed by type:
   - **ResponseTypeText**: Display as assistant message
   - **ResponseTypeAction**: Execute tool via ToolRouter
   - **ResponseTypeError**: Display error to user
7. For tool actions:
   - Show "[Executing tool: name]" immediately
   - Execute asynchronously via `executeTool()` command
   - Display result with ✓ (success) or ✗ (failure)
   - Add result to chat history

This ensures:
- ✅ Permission tracking across TUI and CLI modes
- ✅ Message history persistence
- ✅ Audit logging
- ✅ Self-model enforcement
- ✅ Real-time LLM response streaming
- ✅ Async non-blocking UI updates
- ✅ Tool execution with feedback
- ✅ Performance optimization (100 message limit)

## Development Phases

### ✅ Phase 1: Framework Setup
- Installed Bubble Tea, Lipgloss, Bubbles
- Created TUI package structure

### ✅ Phase 2: Chat Interface
- Built viewport for message history
- Added textarea for input
- Implemented styling and controls

### ✅ Phase 3: Mode Detection
- Added `--headless` flag
- Integrated with chat session
- Mode routing in root command

### ✅ Phase 4: Testing & Documentation
- Build verification
- Usage documentation
- Updated help text
- Unit tests for TUI components

### ✅ Phase 5: LLM Streaming
- **Real-time LLM response streaming**
- Async command pattern with Bubble Tea
- Progressive message display
- Error handling for LLM failures
- Session integration (AddUserMessage/AddAssistantTextMessage)
- Loading indicators ("Thinking..." + streaming cursor ▊)

### ✅ Phase 6: Polish & Finalize
- **Tool execution in TUI context**
  - Detects ResponseTypeAction from LLM
  - Executes tools via ToolRouter.Handle()
  - Displays tool results with ✓ or ✗ indicators
  - Error handling for tool failures
- **Performance optimization**
  - Limits viewport to last 100 messages
  - Prevents memory issues with large histories
  - Shows "(N earlier messages hidden)" indicator
- **Enhanced error handling**
  - Truncates long error messages (80 chars)
  - Graceful degradation for missing session
  - User-friendly error messages
- **Additional tests**
  - TestToolExecutionMessage
  - TestToolExecutionError
  - 9/9 tests passing
- **Code quality**
  - Clean imports and structure
  - Comprehensive documentation
  - Production-ready

## Testing

### Manual Testing

1. **Start TUI:**
   ```bash
   ./bin/goshi
   ```

2. **Type a message** and press `Ctrl+S` to send

3. **Scroll through history** with arrow keys
The TUI uses Bubble Tea's testable architecture with comprehensive unit tests:

**Current Test Coverage:**
- `TestNewModel` - Model initialization
- `TestModelInit` - Init command behavior
- `TestModelQuitOnEscape` - Quit on Esc key
- `TestWindowSizeUpdate` - Window resize handling  
- `TestRenderHeader` - Header rendering
- `TestLLMCompleteMessage` - LLM response completion
- `TestLLMErrorMessage` - LLM error handling
- `TestToolExecutionMessage` - Tool execution success
- `TestToolExecutionError` - Tool execution failure

**All 9/9 tests passing**

Run tests:
```bash
go test ./internal/tui -v
```
   ```bash
   echo "Hello" | ./bin/goshi --headless
   ```

### Automated Testing

Currently, the TUI uses Bubble Tea's testable architecture. Future test coverage will include:
- Unit tests for model state transitions
- Integration tests for chat session interaction
- End-to-end TUI flow tests

## Troubleshooting

### TUI doesn't display correctly

**Issue:** Garbled or misaligned output

**Solutions:**
- Ensure terminal supports ANSI colors and escape sequences
- Try resizing terminal window
- Use a modern terminal emulator (iTerm2, Alacritty, Windows Terminal)

### Keyboard shortcuts not working

**Issue:** Ctrl+S doesn't send messages

**Solutions:**
- Check if terminal is capturing the shortcut
- Try different terminal emulator
- Use `--headless` mode as fallback

### TUI crashes or exits immediately

**Issue:** TUI exits without showing interface

**Solutions:**
- Check that ollama is running: `ollama list`
- Verify goshi.self.model.yaml exists
- Review error output: `./bin/goshi 2>error.log`

## Configuration

TUI respects existing Goshi configuration:

```yaml
# config.yaml
llm_provider: "ollama"
model: "qwen2.5-coder:7b"

safety:
  auto_confirm_permissions: false  # Still prompts in TUI
  require_git_clean: true
```

## Future Enhancements

Planned improvements for the TUI:

- [ ] Real-time streaming output (Phase 5)
- [ ] Tool execution progress indicators
- [ ] Command palette (Ctrl+P)
- [ ] Search through history (Ctrl+F)
- [ ] Multi-session tabs
- [ ] Theme customization
- [ ] Mouse support enhancements
- [ ] Copy/paste improvements
- [ ] Markdown rendering in chat

## Contributing

To contribute TUI improvements:

1. Understand the Bubble Tea architecture
2. Follow the existing Model-Update-View pattern
3. Test both TUI and headless modes
4. Update this documentation
5. Submit PR with clear description

## References

- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [Lipgloss Styling](https://github.com/charmbracelet/lipgloss)
- [Bubbles Components](https://github.com/charmbracelet/bubbles)
- [Elm Architecture](https://guide.elm-lang.org/architecture/)

---

**Last Updated:** Phase 4 - February 10, 2026
