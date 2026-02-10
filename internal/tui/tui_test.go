package tui

import (
	"fmt"
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

func TestLLMCompleteMessage(t *testing.T) {
	m := newModel("test", nil)
	m.ready = true
	m.streaming = true

	// Add a message in progress
	m.messages = append(m.messages, Message{
		Role:       "assistant",
		Content:    "",
		InProgress: true,
	})

	// Simulate LLM completion
	completeMsg := llmCompleteMsg{
		fullResponse: "Test response from LLM",
	}

	updatedModel, _ := m.Update(completeMsg)
	updated := updatedModel.(model)

	if updated.streaming {
		t.Error("expected streaming to be false after completion")
	}

	if updated.statusLine != "Ready" {
		t.Errorf("expected status 'Ready', got %q", updated.statusLine)
	}

	if len(updated.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(updated.messages))
	}

	if updated.messages[0].InProgress {
		t.Error("expected message InProgress to be false")
	}

	if updated.messages[0].Content != "Test response from LLM" {
		t.Errorf("expected content 'Test response from LLM', got %q", updated.messages[0].Content)
	}
}

func TestLLMErrorMessage(t *testing.T) {
	m := newModel("test", nil)
	m.ready = true
	m.streaming = true

	// Add a message in progress
	m.messages = append(m.messages, Message{
		Role:       "assistant",
		Content:    "partial",
		InProgress: true,
	})

	// Simulate LLM error
	errMsg := llmErrorMsg{
		err: fmt.Errorf("test error"),
	}

	updatedModel, _ := m.Update(errMsg)
	updated := updatedModel.(model)

	if updated.streaming {
		t.Error("expected streaming to be false after error")
	}

	if updated.err == nil {
		t.Error("expected error to be set")
	}

	if len(updated.messages) != 0 {
		t.Errorf("expected in-progress message to be removed, got %d messages", len(updated.messages))
	}
}

func TestToolExecutionMessage(t *testing.T) {
	m := newModel("test", nil)
	m.ready = true

	// Simulate tool execution result
	toolMsg := toolExecutionMsg{
		toolName: "fs.read",
		result: map[string]any{
			"result": "file contents here",
		},
	}

	updatedModel, _ := m.Update(toolMsg)
	updated := updatedModel.(model)

	if updated.statusLine != "Ready" {
		t.Errorf("expected status 'Ready', got %q", updated.statusLine)
	}

	if len(updated.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(updated.messages))
	}

	if updated.messages[0].Role != "assistant" {
		t.Errorf("expected role 'assistant', got %q", updated.messages[0].Role)
	}

	if !strings.Contains(updated.messages[0].Content, "fs.read") {
		t.Error("expected message to contain tool name")
	}

	if !strings.Contains(updated.messages[0].Content, "file contents here") {
		t.Error("expected message to contain result")
	}
}

func TestToolExecutionError(t *testing.T) {
	m := newModel("test", nil)
	m.ready = true

	// Simulate tool execution error
	toolMsg := toolExecutionMsg{
		toolName: "fs.write",
		result: map[string]any{
			"error": "permission denied",
		},
	}

	updatedModel, _ := m.Update(toolMsg)
	updated := updatedModel.(model)

	if updated.err == nil {
		t.Error("expected error to be set")
	}

	if len(updated.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(updated.messages))
	}

	if !strings.Contains(updated.messages[0].Content, "failed") {
		t.Error("expected message to indicate failure")
	}

	if !strings.Contains(updated.messages[0].Content, "permission denied") {
		t.Error("expected message to contain error")
	}
}
