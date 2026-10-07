package tui

import (
	"crypto/sha256"
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
	"github.com/mattn/go-runewidth"
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
		taCmd   tea.Cmd
		vpCmd   tea.Cmd
		helpCmd tea.Cmd
	)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlQ:
			return m, tea.Quit
		case tea.KeyCtrlH:
			m.helpPanelVisible = !m.helpPanelVisible
			return m, nil
		}

		if m.helpPanelVisible {
			helpCmd = m.helpPanel.Update(msg)
			return m, helpCmd
		}
	}

	if !m.helpPanelVisible {
		m.textarea, taCmd = m.textarea.Update(msg)
	}

	// Route viewport/scrolling updates based on focused region
	if m.focusedRegion == FocusOutputStream {
		// Output stream is focused - handle viewport scrolling
		m.viewport, vpCmd = m.viewport.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// Send message only when focused on input
			if m.focusedRegion == FocusInput {
				return m.handleSendMessage()
			}
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
			// Cycle focus forward (output <-> input)
			cycle := []FocusRegion{FocusOutputStream, FocusInput}
			for i, region := range cycle {
				if region == m.focusedRegion {
					m.focusedRegion = cycle[(i+1)%len(cycle)]
					break
				}
			}
			return m, nil
		case tea.KeyShiftTab:
			// Cycle focus backward (output <-> input)
			cycle := []FocusRegion{FocusOutputStream, FocusInput}
			for i, region := range cycle {
				if region == m.focusedRegion {
					idx := (i - 1 + len(cycle)) % len(cycle)
					m.focusedRegion = cycle[idx]
					break
				}
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		// Recalculate layout
		m.layout.Recalculate(msg.Width, msg.Height)

		// Update viewport dimensions
		contentWidth := m.layout.OutputStreamWidth - 4
		if contentWidth < 10 {
			contentWidth = 10
		}
		m.viewport.Width = contentWidth
		m.viewport.Height = m.layout.OutputStreamHeight - 2

		// Update textarea dimensions
		m.textarea.SetWidth(m.layout.OutputStreamWidth - 4)

		// Update help panel dimensions
		m.helpPanel.SetSize(msg.Width-4, msg.Height-8)

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

	return m, tea.Batch(taCmd, vpCmd)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing Goshi TUI..."
	}

	// Update status bar metrics
	metrics := selfmodel.ComputeLawMetrics(m.systemPrompt)
	m.statusBar.UpdateMetrics(metrics.RuleLines, metrics.ConstraintCount)
	m.statusBar.UpdateMode(m.mode.String())

	// Update inspect panel metrics (used for top bar content)
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

	// Render output stream (full width)
	outputStream := m.renderOutputStream()
	// Render top bar (two lines)
	topBar := m.renderTopBar(metrics)

	// Build the full view
	var mainContent string
	if m.helpPanelVisible {
		// Show help panel instead of other panels
		helpPanel := ""
		if m.helpPanel.ready {
			helpPanel = m.helpPanel.Render()
		}
		mainContent = helpPanel
	} else {
		mainContent = lipgloss.JoinVertical(
			lipgloss.Left,
			topBar,
			outputStream,
		)
	}

	// Render input area
	inputArea := m.renderInput()

	// Render status bar (single line)
	statusBar := m.statusBar.Render(m.layout.TerminalWidth)

	// Combine vertically
	return lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		inputArea,
		statusBar,
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
	contentWidth := m.viewport.Width
	if contentWidth < 10 {
		contentWidth = 10
	}
	messageWidth := contentWidth - 2
	if messageWidth < 10 {
		messageWidth = 10
	}

	welcomeText := "Welcome to Goshi TUI\n\nCommands:\n  Enter - Send message\n  Ctrl+C/Ctrl+Q - Quit\n  ↑/↓ - Scroll chat"
	sb.WriteString(wrapAndStyle(welcomeText, contentWidth, welcomeStyle))
	sb.WriteString("\n\n")

	if m.streaming {
		sb.WriteString(wrapAndStyle("✨ Streaming response...", contentWidth, statusStyle))
		sb.WriteString("\n\n")
	}

	// Performance optimization: limit displayed messages to last 100
	// to prevent memory issues and slow rendering with large histories
	const maxDisplayMessages = 100
	startIdx := 0
	if len(m.messages) > maxDisplayMessages {
		startIdx = len(m.messages) - maxDisplayMessages
		hidden := fmt.Sprintf("... (%d earlier messages hidden) ...", startIdx)
		sb.WriteString(wrapAndStyle(hidden, contentWidth, statusStyle))
		sb.WriteString("\n\n")
	}

	for i := startIdx; i < len(m.messages); i++ {
		msg := m.messages[i]
		content := msg.Content
		if msg.InProgress {
			content += "▊" // Show cursor for streaming
		}

		sb.WriteString(formatWrappedMessage(msg.Role, content, messageWidth))
		sb.WriteString("\n\n")
	}

	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

func (m model) renderInput() string {
	return fmt.Sprintf(
		"┌─\n%s",
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
		Width(m.layout.OutputStreamWidth).
		Height(m.layout.OutputStreamHeight)

	if m.focusedRegion == FocusOutputStream {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("12"))
	}

	// Content is the viewport
	content := m.viewport.View()

	return borderStyle.Render(content)
}

func (m model) renderTopBar(metrics selfmodel.LawMetrics) string {
	width := m.layout.TerminalWidth
	if width <= 0 {
		return ""
	}

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	positiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	negativeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	guardText := "OFF"
	guardStyle := valueStyle
	if m.inspectPanel.GuardrailsOn() {
		guardText = "ON"
		guardStyle = positiveStyle
	}

	memPercent := 0.0
	if m.telemetry.MemoryMax > 0 {
		memPercent = (float64(m.telemetry.MemoryEntries) / float64(m.telemetry.MemoryMax)) * 100
	}

	hash := sha256.Sum256([]byte(m.systemPrompt))
	sessionID := fmt.Sprintf("%X", hash[:3])

	capabilities := m.inspectPanel.Capabilities()
	toolsValue := "disabled"
	toolsStyle := negativeStyle
	fsValue := "denied"
	fsStyle := negativeStyle
	netValue := "denied"
	netStyle := negativeStyle
	if capabilities != nil {
		if capabilities.ToolsEnabled {
			toolsValue = "enabled"
			toolsStyle = positiveStyle
		}
		fsValue = capabilities.FilesystemStatus
		if capabilities.FilesystemAllowed {
			fsStyle = positiveStyle
		}
		netValue = capabilities.NetworkStatus
		if capabilities.NetworkAllowed {
			netStyle = positiveStyle
		}
	}

	line1 := joinSegments([]string{
		formatTopBarKV("Session", sessionID, labelStyle, valueStyle),
		formatTopBarKV("Temp", fmt.Sprintf("%.1f", m.telemetry.Temperature), labelStyle, valueStyle),
		formatTopBarKV("Guard", guardText, labelStyle, guardStyle),
		formatTopBarKV("Laws", fmt.Sprintf("%d", metrics.RuleLines), labelStyle, valueStyle),
		formatTopBarKV("Constraints", fmt.Sprintf("%d", metrics.ConstraintCount), labelStyle, valueStyle),
		formatTopBarKV("Entries", fmt.Sprintf("%d/%d", m.telemetry.MemoryEntries, m.telemetry.MemoryMax), labelStyle, valueStyle),
	}, width)

	line2 := joinSegments([]string{
		formatTopBarKV("Usage", fmt.Sprintf("%.0f%%", memPercent), labelStyle, valueStyle),
		formatTopBarKV("Dry Run", onOff(m.toggles.DryRun), labelStyle, valueStyle),
		formatTopBarKV("Deterministic", onOff(m.toggles.Deterministic), labelStyle, valueStyle),
		formatTopBarKV("Tools", toolsValue, labelStyle, toolsStyle),
		formatTopBarKV("Filesystem", fsValue, labelStyle, fsStyle),
		formatTopBarKV("Network", netValue, labelStyle, netStyle),
	}, width)

	return lipgloss.JoinVertical(lipgloss.Left, line1, line2)
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
func styleUserLine(text string) string { return userStyle.Render(text) }
func styleAssistantMessage(text string) string {
	role := roleStyle.Render("ASSISTANT: ")
	return assistantStyle.Render(role + text)
}
func styleAssistantLine(text string) string { return assistantStyle.Render(text) }
func styleSystemMessage(text string) string {
	role := roleStyle.Render("SYSTEM: ")
	return systemStyle.Render(role + text)
}
func styleSystemLine(text string) string { return systemStyle.Render(text) }
func styleToolMessage(text string) string {
	role := roleStyle.Render("TOOL: ")
	return toolStyle.Render(role + text)
}
func styleToolLine(text string) string { return toolStyle.Render(text) }
func styleStatus(text string) string   { return statusStyle.Render(text) }
func styleError(text string) string    { return errorStyle.Render(text) }
func styleWelcome(text string) string  { return welcomeStyle.Render(text) }

func onOff(value bool) string {
	if value {
		return "ON"
	}
	return "OFF"
}

func formatTopBarKV(label, value string, labelStyle, valueStyle lipgloss.Style) string {
	return labelStyle.Render(label) + " " + valueStyle.Render(value)
}

func joinSegments(segments []string, width int) string {
	if width <= 0 {
		return strings.Join(segments, " │ ")
	}

	sep := " │ "
	sepWidth := lipgloss.Width(sep)
	line := ""
	currentWidth := 0
	for i, seg := range segments {
		segWidth := lipgloss.Width(seg)
		extra := segWidth
		if i > 0 {
			extra += sepWidth
		}
		if currentWidth > 0 && currentWidth+extra > width {
			break
		}
		if i > 0 {
			line += sep
			currentWidth += sepWidth
		}
		line += seg
		currentWidth += segWidth
	}

	if currentWidth < width {
		line += strings.Repeat(" ", width-currentWidth)
	}
	return line
}

func wrapPlainText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	lines := strings.Split(text, "\n")
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			wrapped = append(wrapped, "")
			continue
		}

		indentEnd := 0
		for indentEnd < len(line) {
			if line[indentEnd] == ' ' || line[indentEnd] == '\t' {
				indentEnd++
				continue
			}
			break
		}
		indent := line[:indentEnd]
		rest := strings.TrimLeft(line, " \t")
		if rest == "" {
			wrapped = append(wrapped, indent)
			continue
		}

		indentWidth := runewidth.StringWidth(indent)
		wrapWidth := width - indentWidth
		if wrapWidth < 1 {
			wrapWidth = 1
		}

		current := indent
		currentWidth := indentWidth
		words := strings.Fields(rest)
		for _, word := range words {
			wordWidth := runewidth.StringWidth(word)
			if wordWidth > wrapWidth {
				if currentWidth > indentWidth {
					wrapped = append(wrapped, current)
					current = indent
					currentWidth = indentWidth
				}
				for _, chunk := range splitLongWord(word, wrapWidth) {
					wrapped = append(wrapped, indent+chunk)
				}
				current = indent
				currentWidth = indentWidth
				continue
			}

			if currentWidth > indentWidth && currentWidth+1+wordWidth > width {
				wrapped = append(wrapped, current)
				current = indent + word
				currentWidth = indentWidth + wordWidth
				continue
			}

			if currentWidth == indentWidth {
				current += word
				currentWidth += wordWidth
				continue
			}

			current += " " + word
			currentWidth += 1 + wordWidth
		}

		if current != "" {
			wrapped = append(wrapped, current)
		}
	}

	return wrapped
}

func splitLongWord(word string, width int) []string {
	if width < 1 {
		return []string{word}
	}
	chunks := []string{}
	var buf strings.Builder
	currentWidth := 0
	for _, r := range word {
		w := runewidth.RuneWidth(r)
		if currentWidth+w > width && buf.Len() > 0 {
			chunks = append(chunks, buf.String())
			buf.Reset()
			currentWidth = 0
		}
		buf.WriteRune(r)
		currentWidth += w
	}
	if buf.Len() > 0 {
		chunks = append(chunks, buf.String())
	}
	return chunks
}

func wrapAndStyle(text string, width int, style lipgloss.Style) string {
	lines := wrapPlainText(text, width)
	styled := make([]string, 0, len(lines))
	for _, line := range lines {
		styled = append(styled, style.Render(line))
	}
	return strings.Join(styled, "\n")
}

func formatWrappedMessage(role, content string, width int) string {
	label := "ASSISTANT: "
	styleFn := styleAssistantLine
	switch role {
	case "user":
		label = "USER: "
		styleFn = styleUserLine
	case "system":
		label = "SYSTEM: "
		styleFn = styleSystemLine
	case "tool":
		label = "TOOL: "
		styleFn = styleToolLine
	}

	labelWidth := runewidth.StringWidth(label)
	wrapWidth := width - labelWidth
	if wrapWidth < 1 {
		wrapWidth = 1
	}

	lines := wrapPlainText(content, wrapWidth)
	if len(lines) == 0 {
		lines = []string{""}
	}

	indent := strings.Repeat(" ", labelWidth)
	styled := make([]string, 0, len(lines))
	for i, line := range lines {
		if i == 0 {
			styled = append(styled, styleFn(label+line))
			continue
		}
		styled = append(styled, styleFn(indent+line))
	}

	return strings.Join(styled, "\n")
}
