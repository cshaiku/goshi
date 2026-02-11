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

// InputToggles represents the state of input toggles
type InputToggles struct {
	DryRun        bool
	Deterministic bool
}

// model is the TUI application state
type model struct {
	// Components
	viewport     viewport.Model
	textarea     textarea.Model
	messages     []Message
	inspectPanel *InspectPanel
	auditPanel   *AuditPanel
	helpPanel    *HelpPanel
	statusBar    *StatusBar
	layout       *Layout
	telemetry    *Telemetry

	// State
	ready             bool
	focusedRegion     FocusRegion
	mode              Mode
	toggles           InputToggles
	statusLine        string
	err               error
	auditPanelVisible bool
	auditPanelRefresh int // Counter to refresh audit panel less frequently
	helpPanelVisible  bool

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

	// Set backend and model from session if available
	if sess != nil {
		telemetry.Backend = sess.Provider
		telemetry.ModelName = sess.Model
	} else {
		telemetry.Backend = "ollama"
		telemetry.ModelName = "unknown"
	}

	statusBar := NewStatusBar(telemetry)
	inspectPanel := NewInspectPanel(telemetry)
	helpPanel := NewHelpPanel()
	layout := NewLayout()

	// Initialize audit panel
	auditPanel := NewAuditPanel("")
	if sess != nil && sess.AuditLogger != nil {
		auditPanel = NewAuditPanel(sess.AuditLogger.FilePath())
	}

	return model{
		viewport:          vp,
		textarea:          ta,
		messages:          []Message{},
		inspectPanel:      inspectPanel,
		auditPanel:        auditPanel,
		helpPanel:         helpPanel,
		statusBar:         statusBar,
		layout:            layout,
		telemetry:         telemetry,
		focusedRegion:     FocusInput,
		mode:              ModeChat,
		toggles:           InputToggles{DryRun: false, Deterministic: false},
		chatSession:       sess,
		systemPrompt:      systemPrompt,
		statusLine:        "Ready",
		auditPanelVisible: false,
		helpPanelVisible:  false,
		auditPanelRefresh: 0,
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
	} else if m.focusedRegion == FocusAuditPanel {
		// Audit panel is focused - handle scrolling there
		_ = m.auditPanel.Update(msg)
	} else {
		// Output stream is focused - handle viewport scrolling
		m.viewport, vpCmd = m.viewport.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlQ:
			return m, tea.Quit
		case tea.KeyEnter:
			// Send message only when focused on input
			if m.focusedRegion == FocusInput {
				return m.handleSendMessage()
			}
		case tea.KeyCtrlA:
			// Toggle audit panel
			m.auditPanelVisible = !m.auditPanelVisible
			m.layout.AuditPanelVisible = m.auditPanelVisible
			// Reset focus if toggling off
			if !m.auditPanelVisible && m.focusedRegion == FocusAuditPanel {
				m.focusedRegion = FocusInput
			}
			return m, nil
		case tea.KeyCtrlH:
			// Toggle help panel
			m.helpPanelVisible = !m.helpPanelVisible
			return m, nil
		case tea.KeyCtrlL:
			// Cycle through modes
			m.mode = (m.mode + 1) % 3
			return m, nil
		case tea.KeyCtrlD:
			// Toggle dry run
			m.toggles.DryRun = !m.toggles.DryRun
			return m, nil
		case tea.KeyCtrlT:
			// Toggle deterministic
			m.toggles.Deterministic = !m.toggles.Deterministic
			return m, nil
		case tea.KeyTab:
			// Cycle focus forward (only through visible regions)
			if m.auditPanelVisible {
				// Cycle through all 4 regions: 0->1->2->3->0
				m.focusedRegion = (m.focusedRegion + 1) % 4
			} else {
				// Skip audit panel: cycle 0->1->3->0
				cycle := []FocusRegion{FocusOutputStream, FocusInspectPanel, FocusInput}
				for i, region := range cycle {
					if region == m.focusedRegion {
						m.focusedRegion = cycle[(i+1)%len(cycle)]
						break
					}
				}
			}
			return m, nil
		case tea.KeyShiftTab:
			// Cycle focus backward
			if m.auditPanelVisible {
				// Cycle backward through all 4 regions: 3->2->1->0->3
				m.focusedRegion = (m.focusedRegion + 3) % 4
			} else {
				// Skip audit panel: cycle 0->3->1->0
				cycle := []FocusRegion{FocusOutputStream, FocusInspectPanel, FocusInput}
				for i, region := range cycle {
					if region == m.focusedRegion {
						idx := (i - 1 + len(cycle)) % len(cycle)
						m.focusedRegion = cycle[idx]
						break
					}
				}
			}
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

		// Update help panel dimensions
		m.helpPanel.SetSize(msg.Width-4, msg.Height-8)

		// Update audit panel dimensions if visible
		if m.auditPanelVisible {
			m.auditPanel.SetSize(msg.Width, m.layout.AuditPanelHeight)
		}

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

	// Refresh audit panel occasionally (every 10 updates)
	if m.auditPanelVisible {
		m.auditPanelRefresh++
		if m.auditPanelRefresh >= 10 {
			m.auditPanel.Refresh()
			m.auditPanelRefresh = 0
		}
	}

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

	// Combine output stream and inspect panel horizontally
	topRegion := lipgloss.JoinHorizontal(
		lipgloss.Top,
		outputStream,
		inspectPanel,
	)

	// Build the full view
	var mainContent string
	if m.helpPanelVisible {
		// Show help panel instead of other panels
		helpPanel := ""
		if m.helpPanel.ready {
			helpPanel = m.helpPanel.Render()
		}
		mainContent = helpPanel
	} else if m.auditPanelVisible {
		// Include audit panel
		auditReady := m.auditPanel.ready
		auditPanel := ""
		if auditReady {
			auditPanel = m.auditPanel.Render()
		}

		mainContent = lipgloss.JoinVertical(
			lipgloss.Left,
			topRegion,
			auditPanel,
		)
	} else {
		mainContent = topRegion
	}

	// Render status bar (2 lines)
	statusBar := m.statusBar.Render(m.layout.TerminalWidth)

	// Render input area
	inputArea := m.renderInput()

	// Combine vertically
	return lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
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

	sb.WriteString(styleWelcome("Welcome to Goshi TUI\n\nCommands:\n  Enter - Send message\n  Ctrl+C/Ctrl+Q - Quit\n  ↑/↓ - Scroll chat\n"))
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
	modeDisplay := fmt.Sprintf(" │ Mode: %s (Ctrl+M)", m.mode.String())

	// Toggles display
	dryRunIndicator := ""
	if m.toggles.DryRun {
		dryRunIndicator = " ✓ Dry Run"
	} else {
		dryRunIndicator = " ○ Dry Run"
	}

	deterministic := ""
	if m.toggles.Deterministic {
		deterministic = " ✓ Deterministic"
	} else {
		deterministic = " ○ Deterministic"
	}

	toglesDisplay := fmt.Sprintf(" │ Toggles: %s %s", dryRunIndicator, deterministic)

	// Audit panel indicator
	auditDisplay := ""
	if m.auditPanelVisible {
		auditDisplay = " │ Audit: ✓ (Ctrl+A to hide)"
	} else {
		auditDisplay = " │ Audit: ○ (Ctrl+A to show)"
	}

	return fmt.Sprintf(
		"┌─ Input (Enter: send, Tab: focus, Ctrl+L: mode, Ctrl+D/T: toggle, Ctrl+A: audit, Ctrl+H: help, Ctrl+Q: quit)%s%s%s%s\n%s",
		focusIndicator,
		modeDisplay,
		toglesDisplay,
		auditDisplay,
		m.textarea.View(),
	)
}

// CodeBlock represents a collapsible code block in the output
type CodeBlock struct {
	Language  string
	Content   string
	Collapsed bool // Whether the block is collapsed
	LineCount int
}

// IsCodeBlock checks if text contains a code block marker
func IsCodeBlock(text string) bool {
	return strings.Contains(text, "```")
}

// ExtractCodeBlocks parses code blocks from text
func ExtractCodeBlocks(text string) []CodeBlock {
	var blocks []CodeBlock
	lines := strings.Split(text, "\n")

	var inBlock bool
	var language string
	var content strings.Builder

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if strings.HasPrefix(line, "```") {
			if inBlock {
				// End of block
				lineCount := strings.Count(content.String(), "\n") + 1
				if lineCount > 5 { // Collapse if more than 5 lines
					blocks = append(blocks, CodeBlock{
						Language:  language,
						Content:   content.String(),
						Collapsed: true,
						LineCount: lineCount,
					})
				} else {
					blocks = append(blocks, CodeBlock{
						Language:  language,
						Content:   content.String(),
						Collapsed: false,
						LineCount: lineCount,
					})
				}
				content.Reset()
				inBlock = false
				language = ""
			} else {
				// Start of block
				inBlock = true
				language = strings.TrimPrefix(line, "```")
			}
		} else if inBlock {
			content.WriteString(line + "\n")
		}
	}

	return blocks
}

// RenderCodeBlock renders a code block with collapse support
func (cb *CodeBlock) Render() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	langLabel := ""
	if cb.Language != "" {
		langLabel = fmt.Sprintf(" [%s]", cb.Language)
	}

	if cb.Collapsed {
		// Show collapse indicator with line count, no content
		header := fmt.Sprintf("📦 Code Block%s (%d lines) - Press 'C' to expand", langLabel, cb.LineCount)
		return style.Render(header)
	} else {
		// Show expanded content
		header := fmt.Sprintf("📦 Code Block%s (%d lines) - Press 'C' to collapse", langLabel, cb.LineCount)
		content := strings.TrimRight(cb.Content, "\n") // Remove trailing newline
		return style.Render(header + "\n" + content)
	}
}

// ToggleCollapse toggles the collapsed state
func (cb *CodeBlock) ToggleCollapse() {
	if cb.LineCount > 5 { // Only allow collapsing if more than 5 lines
		cb.Collapsed = !cb.Collapsed
	}
}

// AccessibilityInfo returns screen-reader friendly text for a message
func (msg *Message) AccessibilityInfo() string {
	roleLabel := ""
	switch msg.Role {
	case "user":
		roleLabel = "User message"
	case "assistant":
		roleLabel = "Assistant message"
	case "system":
		roleLabel = "System message"
	case "tool":
		roleLabel = "Tool message"
	default:
		roleLabel = "Message"
	}

	status := ""
	if msg.InProgress {
		status = ", currently streaming"
	}

	return fmt.Sprintf("%s%s: %s", roleLabel, status, msg.Content)
}

// AccessibilityDescription returns a description suitable for ARIA labels
func (m *model) AccessibilityDescription() string {
	return fmt.Sprintf(
		"Goshi TUI. Current mode: %s. Toggles: Dry Run %s, Deterministic %s. "+
			"Focus: Use Tab to cycle between output stream, inspect panel, and input area. "+
			"Commands: Enter to send, Ctrl+L to change mode, Ctrl+D/T to toggle, Ctrl+Q to quit.",
		m.mode.String(),
		func() string {
			if m.toggles.DryRun {
				return "on"
			}
			return "off"
		}(),
		func() string {
			if m.toggles.Deterministic {
				return "on"
			}
			return "off"
		}(),
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
