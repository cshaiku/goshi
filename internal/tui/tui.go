package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Run starts the TUI application
func Run() error {
	p := tea.NewProgram(initialModel())
	_, err := p.Run()
	return err
}

// Model is the TUI model
type model struct {
	ready bool
}

func initialModel() model {
	return model{ready: true}
}

func (m model) Init() tea.Cmd {
return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
return m, nil
}

func (m model) View() string {
return "Goshi TUI - Coming soon!\n\nPress Ctrl+C to quit.\n"
}
