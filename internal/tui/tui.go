package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cshaiku/goshi/internal/cli"
	"github.com/cshaiku/goshi/internal/selfmodel"
)

// Run starts the TUI application
func Run(systemPrompt string, session *cli.ChatSession) error {
	p := tea.NewProgram(
		newModel(systemPrompt, session),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}

// Message represents a chat message
type Message struct {
	Role    string // "user" or "assistant"
	Content string
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
	session      *cli.ChatSession
	systemPrompt string
}

func newModel(systemPrompt string, session *cli.ChatSession) model {
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
		session:      session,
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

func (m model) handleSendMessage() (tea.Model, tea.Cmd) {
	userInput := strings.TrimSpace(m.textarea.Value())
	if userInput == "" {
		return m, nil
	}

	// Add message
	m.messages = append(m.messages, Message{
		Role:    "user",
		Content: userInput,
	})

	m.textarea.Reset()
	m.updateViewportContent()

	// TODO Phase 4: Integrate with LLM
	m.statusLine = "Thinking..."
	m.messages = append(m.messages, Message{
		Role:    "assistant",
		Content: fmt.Sprintf("[Phase 4 TODO: LLM integration] Echo: %s", userInput),
	})
	m.updateViewportContent()
	m.statusLine = "Ready"

	return m, nil
}

func (m *model) updateViewportContent() {
	var sb strings.Builder

	sb.WriteString(styleWelcome("Welcome to Goshi TUI\n\nCommands:\n  Ctrl+S - Send message\n  Ctrl+C/Esc - Quit\n  ↑/↓ - Scroll chat\n"))
	sb.WriteString("\n")

	for _, msg := range m.messages {
		if msg.Role == "user" {
			sb.WriteString(styleUserMessage(msg.Content))
		} else {
			sb.WriteString(styleAssistantMessage(msg.Content))
		}
		sb.WriteString("\n\n")
	}

	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

func (m model) renderHeader() string {
	metrics := selfmodel.ComputeLawMetrics(m.systemPrompt)
	status := "STAGED"

	if m.session != nil && m.session.Permissions != nil {
		perms := m.session.Permissions
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
