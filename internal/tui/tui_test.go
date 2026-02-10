package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModel(t *testing.T) {
	systemPrompt := "Test system prompt"
	m := newModel(systemPrompt, nil)

	if m.systemPrompt != systemPrompt {
		t.Errorf("expected systemPrompt %q, got %q", systemPrompt, m.systemPrompt)
	}

	if m.statusLine != "Ready" {
		t.Errorf("expected initial status 'Ready', got %q", m.statusLine)
	}

	if len(m.messages) != 0 {
		t.Errorf("expected empty messages, got %d messages", len(m.messages))
	}
}

func TestModelInit(t *testing.T) {
	m := newModel("test", nil)
	cmd := m.Init()

	if cmd == nil {
		t.Error("expected non-nil Init command")
	}
}

func TestModelQuitOnEscape(t *testing.T) {
	m := newModel("test", nil)
	m.ready = true

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("expected Quit command on Esc")
	}

	if _, ok := updatedModel.(model); !ok {
		t.Error("expected model type to be returned")
	}
}

func TestWindowSizeUpdate(t *testing.T) {
	m := newModel("test", nil)

	msg := tea.WindowSizeMsg{
		Width:  100,
		Height: 40,
	}

	updatedModel, _ := m.Update(msg)
	updated := updatedModel.(model)

	if updated.width != 100 {
		t.Errorf("expected width 100, got %d", updated.width)
	}

	if updated.height != 40 {
		t.Errorf("expected height 40, got %d", updated.height)
	}

	if !updated.ready {
		t.Error("expected ready=true after window size update")
	}
}

func TestRenderHeader(t *testing.T) {
	systemPrompt := "Test system"
	m := newModel(systemPrompt, nil)
	m.ready = true

	header := m.renderHeader()

	if header == "" {
		t.Error("expected non-empty header")
	}

	if !strings.Contains(header, "GOSHI TUI") {
		t.Error("expected header to contain 'GOSHI TUI'")
	}
}
