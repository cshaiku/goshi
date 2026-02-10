package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	Role       string // "user" or "assistant"
	Content    string
	InProgress bool // True if still streaming
}

// model is the TUI application state
type model struct {
	// Components
	viewport viewport.Model
	textarea textarea.Model
	messages []Message

	// State
	ready      bool
	width      int
	height     int
	statusLine string
	err        error

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

	return model{
		viewport:     vp,
		textarea:     ta,
		messages:     []Message{},
		chatSession:  sess,
		systemPrompt: systemPrompt,
		statusLine:   "Ready",
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		taCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, taCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyCtrlS:
			return m.handleSendMessage()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 4
		footerHeight := 5
		statusHeight := 1

		m.viewport.Width = msg.Width - 2
		m.viewport.Height = msg.Height - headerHeight - footerHeight - statusHeight
		m.textarea.SetWidth(msg.Width - 4)

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
				m.messages[len(m.messages)-1].Content = msg.parseResult.Response.Text
				if m.chatSession != nil {
					m.chatSession.AddAssistantTextMessage(msg.parseResult.Response.Text)
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

	var sb strings.Builder

	// Header
	sb.WriteString(m.renderHeader())
	sb.WriteString("\n\n")

	// Chat viewport
	sb.WriteString(m.viewport.View())
	sb.WriteString("\n\n")

	// Status
	sb.WriteString(m.renderStatus())
	sb.WriteString("\n")

	// Input
	sb.WriteString(m.renderInput())

	return sb.String()
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

func (m *model) updateViewportContent() {
	var sb strings.Builder

	sb.WriteString(styleWelcome("Welcome to Goshi TUI\n\nCommands:\n  Ctrl+S - Send message\n  Ctrl+C/Esc - Quit\n  ↑/↓ - Scroll chat\n"))
	sb.WriteString("\n")

	if m.streaming {
		sb.WriteString(styleStatus("✨ Streaming response...\n\n"))
	}

	for _, msg := range m.messages {
		if msg.Role == "user" {
			sb.WriteString(styleUserMessage(msg.Content))
		} else {
			content := msg.Content
			if msg.InProgress {
				content += "▊" // Show cursor for streaming
			}
			sb.WriteString(styleAssistantMessage(content))
		}
		sb.WriteString("\n\n")
	}

	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

func (m model) renderHeader() string {
	metrics := selfmodel.ComputeLawMetrics(m.systemPrompt)
	status := "STAGED"

	if m.chatSession != nil && m.chatSession.Permissions != nil {
		perms := m.chatSession.Permissions
		if perms.FSRead && perms.FSWrite {
			status = "ACTIVE (FS_READ + FS_WRITE)"
		} else if perms.FSRead {
			status = "ACTIVE (FS_READ)"
		} else if perms.FSWrite {
			status = "ACTIVE (FS_WRITE)"
		}
	}

	return styleHeader(fmt.Sprintf(
		"╔═ GOSHI TUI ════════════════════════════════════════════╗\n"+
			"║ Laws: %d lines │ Constraints: %d │ Status: %s",
		metrics.RuleLines,
		metrics.ConstraintCount,
		status,
	))
}

func (m model) renderStatus() string {
	text := fmt.Sprintf("─ %s ", m.statusLine)
	if m.err != nil {
		return styleError(fmt.Sprintf("─ Error: %v ", m.err))
	}
	return styleStatus(text)
}

func (m model) renderInput() string {
	return fmt.Sprintf(
		"┌─ Input (Ctrl+S to send)\n%s",
		m.textarea.View(),
	)
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

func styleHeader(text string) string      { return headerStyle.Render(text) }
func styleUserMessage(text string) string { return userStyle.Render("You: " + text) }
func styleAssistantMessage(text string) string {
	return assistantStyle.Render("Goshi: " + text)
}
func styleStatus(text string) string  { return statusStyle.Render(text) }
func styleError(text string) string   { return errorStyle.Render(text) }
func styleWelcome(text string) string { return welcomeStyle.Render(text) }
