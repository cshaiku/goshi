package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

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

	if updated.layout.TerminalWidth != 100 {
		t.Errorf("expected terminal width 100, got %d", updated.layout.TerminalWidth)
	}

	if updated.layout.TerminalHeight != 40 {
		t.Errorf("expected terminal height 40, got %d", updated.layout.TerminalHeight)
	}

	if !updated.ready {
		t.Error("expected ready=true after window size update")
	}
}

func TestStatusBar(t *testing.T) {
	systemPrompt := "Test system"
	m := newModel(systemPrompt, nil)
	m.ready = true

	// Trigger window size to initialize layout
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})

	view := m.View()

	if view == "" {
		t.Error("expected non-empty view")
	}

	if !strings.Contains(view, "goshi") {
		t.Error("expected view to contain 'goshi'")
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

// Phase 1: Layout and Infrastructure Tests

func TestLayoutCalculations(t *testing.T) {
	layout := NewLayout()
	layout.Recalculate(100, 40)

	if layout.TerminalWidth != 100 {
		t.Errorf("expected terminal width 100, got %d", layout.TerminalWidth)
	}

	if layout.TerminalHeight != 40 {
		t.Errorf("expected terminal height 40, got %d", layout.TerminalHeight)
	}

	// Check split ratio (70/30)
	expectedOutput := int(float64(100) * 0.70)
	if layout.OutputStreamWidth != expectedOutput {
		t.Errorf("expected output width %d, got %d", expectedOutput, layout.OutputStreamWidth)
	}

	expectedPanel := 100 - expectedOutput
	if layout.InspectPanelWidth != expectedPanel {
		t.Errorf("expected panel width %d, got %d", expectedPanel, layout.InspectPanelWidth)
	}
}

func TestLayoutMinimumSize(t *testing.T) {
	layout := NewLayout()
	minWidth, minHeight := layout.MinimumSize()

	if minWidth != 80 || minHeight != 24 {
		t.Errorf("expected minimum size (80, 24), got (%d, %d)", minWidth, minHeight)
	}
}

func TestTelemetryRecordRequest(t *testing.T) {
	telemetry := NewTelemetry()

	latency := 100 * time.Millisecond
	telemetry.RecordRequest(latency, 500, 0.05)

	if telemetry.LastLatency.Milliseconds() != 100 {
		t.Errorf("expected latency 100ms, got %dms", telemetry.LastLatency.Milliseconds())
	}

	if telemetry.TokensUsed != 500 {
		t.Errorf("expected tokens used 500, got %d", telemetry.TokensUsed)
	}

	if telemetry.SessionCost != 0.05 {
		t.Errorf("expected session cost 0.05, got %f", telemetry.SessionCost)
	}

	if telemetry.RequestCount != 1 {
		t.Errorf("expected request count 1, got %d", telemetry.RequestCount)
	}
}

func TestTelemetryMemoryTracking(t *testing.T) {
	telemetry := NewTelemetry()
	telemetry.UpdateMemory(42)

	if telemetry.MemoryEntries != 42 {
		t.Errorf("expected memory entries 42, got %d", telemetry.MemoryEntries)
	}
}

func TestStatusBarRender(t *testing.T) {
	telemetry := NewTelemetry()
	telemetry.Backend = "ollama"
	telemetry.ModelName = "test-model"

	statusBar := NewStatusBar(telemetry)
	statusBar.UpdateMetrics(312, 9)

	rendered := statusBar.Render(100)

	if rendered == "" {
		t.Error("expected non-empty status bar")
	}

	if !strings.Contains(rendered, "goshi") {
		t.Error("expected status bar to contain 'goshi'")
	}

	if !strings.Contains(rendered, "Laws: 312") {
		t.Error("expected status bar to contain law count")
	}

	if !strings.Contains(rendered, "C: 9") {
		t.Error("expected status bar to contain constraint count")
	}

	if !strings.Contains(rendered, "ollama") {
		t.Error("expected status bar to contain backend name")
	}

	if !strings.Contains(rendered, "test-model") {
		t.Error("expected status bar to contain model name")
	}
}

func TestInspectPanelStub(t *testing.T) {
	panel := NewInspectPanel()
	panel.SetSize(30, 20)

	rendered := panel.Render()

	if rendered == "" {
		t.Error("expected non-empty inspect panel")
	}

	if !strings.Contains(rendered, "INSPECT PANEL") {
		t.Error("expected panel to contain header")
	}

	if !strings.Contains(rendered, "Phase 2") {
		t.Error("expected panel to indicate Phase 2 placeholder")
	}
}

func TestFocusCycling(t *testing.T) {
	m := newModel("test", nil)
	m.ready = true

	// Initial focus should be on input
	if m.focusedRegion != FocusInput {
		t.Errorf("expected initial focus on input, got %d", m.focusedRegion)
	}

	// Pressing Tab should cycle forward
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)

	if m.focusedRegion != FocusOutputStream {
		t.Errorf("expected focus on output stream after Tab, got %d", m.focusedRegion)
	}

	// Pressing Tab again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)

	if m.focusedRegion != FocusInspectPanel {
		t.Errorf("expected focus on inspect panel after second Tab, got %d", m.focusedRegion)
	}

	// Pressing Tab again should wrap back to input
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)

	if m.focusedRegion != FocusInput {
		t.Errorf("expected focus to wrap back to input, got %d", m.focusedRegion)
	}
}

func TestComponentsInitialized(t *testing.T) {
	m := newModel("test", nil)

	if m.layout == nil {
		t.Error("expected layout to be initialized")
	}

	if m.telemetry == nil {
		t.Error("expected telemetry to be initialized")
	}

	if m.statusBar == nil {
		t.Error("expected status bar to be initialized")
	}

	if m.inspectPanel == nil {
		t.Error("expected inspect panel to be initialized")
	}
}
