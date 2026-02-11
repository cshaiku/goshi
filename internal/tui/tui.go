package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cshaiku/goshi/internal/app"
	"github.com/cshaiku/goshi/internal/llm"
	"github.com/cshaiku/goshi/internal/selfmodel"
	"github.com/cshaiku/goshi/internal/session"
)

// Run starts the TUI application
func Run(systemPrompt string, sess *session.ChatSession) error {
	p := tea.NewProgram(
		newModel(systemPrompt, sess),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}

// Message represents a chat message
type Message struct {
	Role       string // "user", "assistant", "system", or "tool"
	Content    string
	InProgress bool // True if still streaming
}

// Mode represents the TUI operational mode
type Mode int

const (
	ModeChat Mode = iota
	ModeCommand
	ModeDiff
)

// String returns the string representation of the mode
func (m Mode) String() string {
	switch m {
	case ModeChat:
		return "Chat"
	case ModeCommand:
		return "Command"
	case ModeDiff:
		return "Diff"
	default:
		return "Chat"
	}
}

// model is the TUI application state
type model struct {
	// Components
	viewport     viewport.Model
	textarea     textarea.Model
	messages     []Message
	inspectPanel *InspectPanel
	statusBar    *StatusBar
	layout       *Layout
	telemetry    *Telemetry

	// State
	ready         bool
	focusedRegion FocusRegion
	mode          Mode
	statusLine    string
	err           error

	// Integration
	chatSession  *session.ChatSession
	systemPrompt string

	// Streaming state
	streaming bool
}

func newModel(systemPrompt string, sess *session.ChatSession) model {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.Prompt = "│ "
	ta.CharLimit = 4000
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	vp := viewport.New(80, 20)

	// Initialize new components
	telemetry := NewTelemetry()
	telemetry.Backend = "ollama"
	telemetry.ModelName = "unknown"

	statusBar := NewStatusBar(telemetry)
	inspectPanel := NewInspectPanel(telemetry)
	layout := NewLayout()

	return model{
		viewport:      vp,
		textarea:      ta,
		messages:      []Message{},
		inspectPanel:  inspectPanel,
		statusBar:     statusBar,
		layout:        layout,
		telemetry:     telemetry,
		focusedRegion: FocusInput,
		mode:          ModeChat,
		chatSession:   sess,
		systemPrompt:  systemPrompt,
		statusLine:    "Ready",
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		taCmd tea.Cmd
		vpCmd tea.Cmd
		ipCmd tea.Cmd
	)

	m.textarea, taCmd = m.textarea.Update(msg)

	// Route viewport/scrolling updates based on focused region
	if m.focusedRegion == FocusInspectPanel {
		// Inspect panel is focused - handle scrolling there
		ipCmd = m.inspectPanel.Update(msg)
	} else {
		// Output stream is focused - handle viewport scrolling
		m.viewport, vpCmd = m.viewport.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyCtrlS:
			return m.handleSendMessage()
		case tea.KeyCtrlM:
			// Cycle through modes
			m.mode = (m.mode + 1) % 3
			return m, nil
		case tea.KeyTab:
			// Cycle focus forward
			m.focusedRegion = (m.focusedRegion + 1) % 3
			return m, nil
		case tea.KeyShiftTab:
			// Cycle focus backward
			m.focusedRegion = (m.focusedRegion + 2) % 3
			return m, nil
		}

	case tea.WindowSizeMsg:
		// Recalculate layout
		m.layout.Recalculate(msg.Width, msg.Height)

		// Update viewport dimensions
		m.viewport.Width = m.layout.OutputStreamWidth - 2
		m.viewport.Height = m.layout.OutputStreamHeight - 2

		// Update textarea dimensions
		m.textarea.SetWidth(m.layout.OutputStreamWidth - 4)

		// Update inspect panel dimensions
		m.inspectPanel.SetSize(m.layout.InspectPanelWidth, m.layout.OutputStreamHeight)

		if !m.ready {
			m.updateViewportContent()
			m.ready = true
		}

	case llmChunkMsg:
		// Update the last message (assistant) with new content
		if len(m.messages) > 0 && m.messages[len(m.messages)-1].InProgress {
			m.messages[len(m.messages)-1].Content += msg.chunk
			m.updateViewportContent()
		}
		return m, nil

	case llmCompleteMsg:
		// Finalize the assistant message
		m.streaming = false
		m.statusLine = "Ready"

		if len(m.messages) > 0 && m.messages[len(m.messages)-1].InProgress {
			m.messages[len(m.messages)-1].InProgress = false

			// Use parsed response if available
			if msg.parseResult != nil && msg.parseResult.Response != nil {
				response := msg.parseResult.Response

				// Handle different response types
				switch response.Type {
				case llm.ResponseTypeAction:
					// Tool execution requested
					if response.Action != nil {
						m.messages[len(m.messages)-1].Content = fmt.Sprintf(
							"[Executing tool: %s]",
							response.Action.Tool,
						)
						m.updateViewportContent()
						return m, executeTool(m.chatSession, response.Action)
					}

				case llm.ResponseTypeText:
					// Regular text response
					m.messages[len(m.messages)-1].Content = response.Text
					if m.chatSession != nil {
						m.chatSession.AddAssistantTextMessage(response.Text)
					}

				case llm.ResponseTypeError:
					// LLM reported an error
					m.messages[len(m.messages)-1].Content = fmt.Sprintf("Error: %s", response.Error)
					m.err = fmt.Errorf("%s", response.Error)
				}
			} else {
				m.messages[len(m.messages)-1].Content = msg.fullResponse
				if m.chatSession != nil {
					m.chatSession.AddAssistantTextMessage(msg.fullResponse)
				}
			}

			m.updateViewportContent()
		}
		return m, nil

	case toolExecutionMsg:
		// Tool execution completed
		m.statusLine = "Ready"

		// Add tool result as a new assistant message
		if resultStr, ok := msg.result["result"].(string); ok {
			m.messages = append(m.messages, Message{
				Role:    "assistant",
				Content: fmt.Sprintf("✓ Tool executed: %s\n\nResult: %s", msg.toolName, resultStr),
			})
		} else if errStr, ok := msg.result["error"].(string); ok {
			m.messages = append(m.messages, Message{
				Role:    "assistant",
				Content: fmt.Sprintf("✗ Tool failed: %s\n\nError: %s", msg.toolName, errStr),
			})
			m.err = fmt.Errorf("%s", errStr)
		} else {
			m.messages = append(m.messages, Message{
				Role:    "assistant",
				Content: fmt.Sprintf("✓ Tool executed: %s\n\nResult: %v", msg.toolName, msg.result),
			})
		}

		m.updateViewportContent()
		return m, nil

	case llmErrorMsg:
		m.streaming = false
		m.err = msg.err
		m.statusLine = "Error"

		// Remove the in-progress message
		if len(m.messages) > 0 && m.messages[len(m.messages)-1].InProgress {
			m.messages = m.messages[:len(m.messages)-1]
		}

		m.updateViewportContent()
		return m, nil

	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(taCmd, vpCmd, ipCmd)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing Goshi TUI..."
	}

	// Update status bar metrics
	metrics := selfmodel.ComputeLawMetrics(m.systemPrompt)
	m.statusBar.UpdateMetrics(metrics.RuleLines, metrics.ConstraintCount)

	// Update inspect panel metrics
	m.inspectPanel.UpdateMetrics(metrics.RuleLines, metrics.ConstraintCount)
	m.inspectPanel.SetGuardrails(true)

	// Update telemetry status and capabilities based on chat session
	if m.chatSession != nil && m.chatSession.Permissions != nil {
		perms := m.chatSession.Permissions
		if perms.FSRead && perms.FSWrite {
			m.telemetry.UpdateStatus("ACTIVE")
		} else if perms.FSRead || perms.FSWrite {
			m.telemetry.UpdateStatus("ACTIVE")
		}

		// Update capabilities
		caps := &Capabilities{
			ToolsEnabled:      true,
			FilesystemAllowed: perms.FSRead || perms.FSWrite,
			FilesystemStatus:  "denied",
			NetworkAllowed:    false,
			NetworkStatus:     "denied",
		}

		if perms.FSRead && perms.FSWrite {
			caps.FilesystemStatus = "allowed"
		} else if perms.FSRead {
			caps.FilesystemStatus = "read-only"
		}

		m.inspectPanel.UpdateCapabilities(caps)
	}

	// Update memory count
	if m.chatSession != nil {
		m.telemetry.UpdateMemory(len(m.chatSession.Messages))
	}

	// Render output stream (left side)
	outputStream := m.renderOutputStream()

	// Render inspect panel (right side) with system prompt
	inspectPanel := m.inspectPanel.Render(m.systemPrompt)

	// Combine horizontally using lipgloss
	topRegion := lipgloss.JoinHorizontal(
		lipgloss.Top,
		outputStream,
		inspectPanel,
	)

	// Render status bar (2 lines)
	statusBar := m.statusBar.Render(m.layout.TerminalWidth)

	// Render input area
	inputArea := m.renderInput()

	// Combine vertically
	return lipgloss.JoinVertical(
		lipgloss.Left,
		topRegion,
		statusBar,
		inputArea,
	)
}

// Custom message types
type errMsg error

type llmChunkMsg struct {
	chunk string
}

type llmCompleteMsg struct {
	fullResponse string
	parseResult  *llm.ParseResult
}

type llmErrorMsg struct {
	err error
}

type toolExecutionMsg struct {
	toolName string
	result   map[string]any
}

func (m model) handleSendMessage() (tea.Model, tea.Cmd) {
	userInput := strings.TrimSpace(m.textarea.Value())
	if userInput == "" {
		return m, nil
	}

	// Don't allow sending while streaming
	if m.streaming {
		return m, nil
	}

	// Add user message to history
	m.messages = append(m.messages, Message{
		Role:    "user",
		Content: userInput,
	})

	// Add to session
	if m.chatSession != nil {
		m.chatSession.AddUserMessage(userInput)
	}

	m.textarea.Reset()
	m.updateViewportContent()

	// Start streaming assistant response
	m.statusLine = "Thinking..."
	m.streaming = true

	// Add placeholder for assistant message
	m.messages = append(m.messages, Message{
		Role:       "assistant",
		Content:    "",
		InProgress: true,
	})
	m.updateViewportContent()

	return m, streamLLMResponse(m.chatSession)
}

// streamLLMResponse creates a command that streams LLM response chunks
func streamLLMResponse(sess *session.ChatSession) tea.Cmd {
	return func() tea.Msg {
		// Get stream from backend
		stream, err := sess.Client.Backend().Stream(
			sess.Context,
			sess.Client.System().Raw(),
			sess.ConvertMessagesToLegacy(),
		)
		if err != nil {
			return llmErrorMsg{err: err}
		}
		defer stream.Close()

		// Collect response
		collector := llm.NewResponseCollector(llm.NewStructuredParser())

		// Stream chunks back to TUI
		for {
			chunk, err := stream.Recv()
			if err != nil {
				// Stream completed
				break
			}
			collector.AddChunk(chunk)

			// TODO: Send individual chunks for progressive display
			// For now, we accumulate and send at end
		}

		// Parse complete response
		fullResponse := collector.GetFullResponse()
		parseResult, _ := collector.Parse()

		return llmCompleteMsg{
			fullResponse: fullResponse,
			parseResult:  parseResult,
		}
	}
}

// executeTool executes a tool call via the ToolRouter
func executeTool(sess *session.ChatSession, action *llm.ActionCall) tea.Cmd {
	return func() tea.Msg {
		if sess == nil || sess.ToolRouter == nil {
			return toolExecutionMsg{
				toolName: action.Tool,
				result: map[string]any{
					"error": "session or tool router not initialized",
				},
			}
		}

		// Execute via ToolRouter
		result := sess.ToolRouter.Handle(app.ToolCall{
			Name: action.Tool,
			Args: action.Args,
		})

		// Convert result to map
		resultMap, ok := result.(map[string]any)
		if !ok {
			resultMap = map[string]any{
				"result": fmt.Sprintf("%v", result),
			}
		}

		return toolExecutionMsg{
			toolName: action.Tool,
			result:   resultMap,
		}
	}
}

func (m *model) updateViewportContent() {
	var sb strings.Builder

	sb.WriteString(styleWelcome("Welcome to Goshi TUI\n\nCommands:\n  Ctrl+S - Send message\n  Ctrl+C/Esc - Quit\n  ↑/↓ - Scroll chat\n"))
	sb.WriteString("\n")

	if m.streaming {
		sb.WriteString(styleStatus("✨ Streaming response...\n\n"))
	}

	// Performance optimization: limit displayed messages to last 100
	// to prevent memory issues and slow rendering with large histories
	const maxDisplayMessages = 100
	startIdx := 0
	if len(m.messages) > maxDisplayMessages {
		startIdx = len(m.messages) - maxDisplayMessages
		sb.WriteString(styleStatus(fmt.Sprintf("... (%d earlier messages hidden) ...\n\n", startIdx)))
	}

	for i := startIdx; i < len(m.messages); i++ {
		msg := m.messages[i]
		content := msg.Content
		if msg.InProgress {
			content += "▊" // Show cursor for streaming
		}

		switch msg.Role {
		case "user":
			sb.WriteString(styleUserMessage(content))
		case "system":
			sb.WriteString(styleSystemMessage(content))
		case "tool":
			sb.WriteString(styleToolMessage(content))
		default: // "assistant" or any other
			sb.WriteString(styleAssistantMessage(content))
		}
		sb.WriteString("\n\n")
	}

	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

func (m model) renderInput() string {
	focusIndicator := ""
	if m.focusedRegion == FocusInput {
		focusIndicator = " (focused)"
	}

	// Mode selector display
	modeDisplay := fmt.Sprintf(" │ Mode: %s (Ctrl+M to cycle)", m.mode.String())

	return fmt.Sprintf(
		"┌─ Input (Ctrl+S to send, Tab to cycle focus)%s%s\n%s",
		focusIndicator,
		modeDisplay,
		m.textarea.View(),
	)
}

// renderOutputStream renders the main output stream (left region)
func (m model) renderOutputStream() string {
	// Create a border with focus indicator
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(m.layout.OutputStreamWidth - 2).
		Height(m.layout.OutputStreamHeight - 2)

	if m.focusedRegion == FocusOutputStream {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("12"))
	}

	// Content is the viewport
	content := m.viewport.View()

	return borderStyle.Render(content)
}

// Styles using lipgloss
var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true)

	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true).
			PaddingLeft(2)

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			PaddingLeft(2)

	systemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			PaddingLeft(2)

	toolStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("141")).
			PaddingLeft(2)

	roleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("250"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	welcomeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
)

func styleHeader(text string) string { return headerStyle.Render(text) }
func styleUserMessage(text string) string {
	role := roleStyle.Render("USER: ")
	return userStyle.Render(role + text)
}
func styleAssistantMessage(text string) string {
	role := roleStyle.Render("ASSISTANT: ")
	return assistantStyle.Render(role + text)
}
func styleSystemMessage(text string) string {
	role := roleStyle.Render("SYSTEM: ")
	return systemStyle.Render(role + text)
}
func styleToolMessage(text string) string {
	role := roleStyle.Render("TOOL: ")
	return toolStyle.Render(role + text)
}
func styleStatus(text string) string  { return statusStyle.Render(text) }
func styleError(text string) string   { return errorStyle.Render(text) }
func styleWelcome(text string) string { return welcomeStyle.Render(text) }
