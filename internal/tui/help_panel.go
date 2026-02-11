package tui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpPanel displays keyboard shortcuts
type HelpPanel struct {
	width    int
	height   int
	viewport viewport.Model
	content  string
	ready    bool
}

// NewHelpPanel creates a new help panel
func NewHelpPanel() *HelpPanel {
	vp := viewport.New(30, 20)
	panel := &HelpPanel{
		viewport: vp,
		ready:    false,
	}
	panel.updateContent()
	return panel
}

// updateContent renders the help text
func (p *HelpPanel) updateContent() {
	p.content = `╔═══════════════════════════════════════╗
║      KEYBOARD SHORTCUTS - HELP        ║
╚═══════════════════════════════════════╝

SENDING & INPUT:
  Enter              - Send message
  Shift+Enter        - New line in input
  Ctrl+Enter         - New line in input
  Tab                - Cycle focus (output/inspect/input)
  Shift+Tab          - Cycle focus backward

MODE & TOGGLES:
  Ctrl+L             - Cycle mode (Chat/Command/Diff)
  Ctrl+D             - Toggle dry run
  Ctrl+T             - Toggle deterministic mode

PANELS & VIEWS:
  Ctrl+A             - Toggle audit panel
  Ctrl+H             - Toggle this help panel

QUIT:
  Ctrl+Q             - Quit application
  Ctrl+C             - Quit application

SHORTCUTS IN DIFFERENT CONTEXTS:
  • When focused on output: ↑/↓ scrolls
  • When focused on inspect: ↑/↓ scrolls
  • When focused on input: text editing + shortcuts above`

	p.viewport.SetContent(p.content)
}

// SetSize updates the panel size
func (p *HelpPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.viewport.Width = width
	p.viewport.Height = height
	// Update content after size changes
	p.updateContent()
	p.ready = true
}

// Update handles messages
func (p *HelpPanel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			if p.viewport.YOffset > 0 {
				p.viewport.YOffset--
			}
		case tea.KeyDown:
			maxScroll := len(p.content) - p.viewport.Height
			if p.viewport.YOffset < maxScroll {
				p.viewport.YOffset++
			}
		case tea.KeyPgUp:
			p.viewport.YOffset = max(0, p.viewport.YOffset-p.viewport.Height)
		case tea.KeyPgDown:
			maxScroll := len(p.content) - p.viewport.Height
			p.viewport.YOffset = min(maxScroll, p.viewport.YOffset+p.viewport.Height)
		case tea.KeyHome:
			p.viewport.YOffset = 0
		case tea.KeyEnd:
			maxScroll := len(p.content) - p.viewport.Height
			p.viewport.YOffset = maxScroll
		}
	case tea.WindowSizeMsg:
		p.SetSize(msg.Width, msg.Height)
	}
	return nil
}

// Render returns the rendered content
func (p *HelpPanel) Render() string {
	if !p.ready {
		return ""
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("5")).
		Padding(0, 1)

	return style.Render(
		lipgloss.NewStyle().
			Width(p.width - 2).
			Height(p.height).
			Render(p.viewport.View()),
	)
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
