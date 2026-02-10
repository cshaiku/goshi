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

The TUI integrates with Goshi's existing chat session management:

```go
// TUI mode initialization
session, _ := session.NewChatSession(ctx, systemPrompt, backend)
tui.Run(systemPrompt, session)
```

This ensures:
- ✅ Permission tracking across TUI and CLI modes
- ✅ Message history persistence
- ✅ Audit logging
- ✅ Self-model enforcement

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

### 🔄 Phase 4: Testing & Documentation (Current)
- Build verification
- Usage documentation
- Updated help text

### 📋 Phase 5: LLM Streaming (Planned)
- Real-time streaming integration
- Tool execution in TUI
- Loading indicators

### 📋 Phase 6: Polish (Planned)
- Error handling improvements
- Performance optimization
- Final testing

## Testing

### Manual Testing

1. **Start TUI:**
   ```bash
   ./bin/goshi
   ```

2. **Type a message** and press `Ctrl+S` to send

3. **Scroll through history** with arrow keys

4. **Quit** with `Ctrl+C` or `Esc`

5. **Test headless mode:**
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
