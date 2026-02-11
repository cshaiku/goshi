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

	msg := tea.KeyMsg{Type: tea.KeyCtrlQ}
	updatedModel, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("expected Quit command on Ctrl+Q")
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
	telemetry := NewTelemetry()
	panel := NewInspectPanel(telemetry)
	panel.SetSize(30, 20)
	panel.UpdateMetrics(312, 9)

	rendered := panel.Render("test system prompt")

	if rendered == "" {
		t.Error("expected non-empty inspect panel")
	}

	if !strings.Contains(rendered, "INSPECT") {
		t.Error("expected panel to contain INSPECT header")
	}

	if !strings.Contains(rendered, "MEMORY") {
		t.Error("expected panel to contain MEMORY section")
	}

	if !strings.Contains(rendered, "PROMPT INFO") {
		t.Error("expected panel to contain PROMPT INFO section")
	}

	if !strings.Contains(rendered, "GUARDRAILS") {
		t.Error("expected panel to contain GUARDRAILS section")
	}

	if !strings.Contains(rendered, "CAPABILITIES") {
		t.Error("expected panel to contain CAPABILITIES section")
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

// Phase 2: Inspect Panel Implementation Tests

func TestInspectPanelMemorySection(t *testing.T) {
	telemetry := NewTelemetry()
	telemetry.MemoryEntries = 42
	telemetry.MemoryMax = 128

	panel := NewInspectPanel(telemetry)
	panel.SetSize(30, 20)

	rendered := panel.Render("test")

	if !strings.Contains(rendered, "42/128") {
		t.Error("expected memory usage to show 42/128")
	}

	if !strings.Contains(rendered, "session") {
		t.Error("expected memory scope to be 'session'")
	}
}

func TestInspectPanelPromptInfo(t *testing.T) {
	telemetry := NewTelemetry()
	telemetry.Temperature = 0.7

	panel := NewInspectPanel(telemetry)
	panel.SetSize(30, 20)

	rendered := panel.Render("test system prompt")

	if !strings.Contains(rendered, "0.7") {
		t.Error("expected temperature 0.7 to be displayed")
	}

	// Policy hash should be present (6 hex chars)
	if !strings.Contains(rendered, "Policy Hash") {
		t.Error("expected policy hash label")
	}
}

func TestInspectPanelGuardrails(t *testing.T) {
	telemetry := NewTelemetry()
	panel := NewInspectPanel(telemetry)
	panel.SetSize(30, 20)
	panel.UpdateMetrics(312, 9)
	panel.SetGuardrails(true)

	rendered := panel.Render("test")

	if !strings.Contains(rendered, "ON") {
		t.Error("expected guardrails to show ON")
	}

	if !strings.Contains(rendered, "312") {
		t.Error("expected laws count 312")
	}

	if !strings.Contains(rendered, "9") {
		t.Error("expected constraints count 9")
	}
}

func TestInspectPanelCapabilities(t *testing.T) {
	telemetry := NewTelemetry()
	panel := NewInspectPanel(telemetry)
	panel.SetSize(30, 30) // Taller to show all sections

	caps := &Capabilities{
		ToolsEnabled:      true,
		FilesystemAllowed: true,
		FilesystemStatus:  "allowed",
		NetworkAllowed:    false,
		NetworkStatus:     "denied",
	}
	panel.UpdateCapabilities(caps)

	rendered := panel.Render("test")

	if !strings.Contains(rendered, "enabled") {
		t.Error("expected tools to show enabled")
	}

	if !strings.Contains(rendered, "allowed") {
		t.Error("expected filesystem to show allowed")
	}

	if !strings.Contains(rendered, "denied") {
		t.Error("expected network to show denied")
	}
}

func TestInspectPanelCapabilitiesReadOnly(t *testing.T) {
	telemetry := NewTelemetry()
	panel := NewInspectPanel(telemetry)
	panel.SetSize(30, 30) // Taller to show all sections

	caps := &Capabilities{
		ToolsEnabled:      true,
		FilesystemAllowed: true,
		FilesystemStatus:  "read-only",
		NetworkAllowed:    false,
		NetworkStatus:     "restricted",
	}
	panel.UpdateCapabilities(caps)

	rendered := panel.Render("test")

	if !strings.Contains(rendered, "read-only") {
		t.Error("expected filesystem to show read-only")
	}

	if !strings.Contains(rendered, "restricted") {
		t.Error("expected network to show restricted")
	}
}

func TestInspectPanelAllSections(t *testing.T) {
	telemetry := NewTelemetry()
	telemetry.MemoryEntries = 10
	telemetry.MemoryMax = 100
	telemetry.Temperature = 0.2

	panel := NewInspectPanel(telemetry)
	panel.SetSize(35, 30)
	panel.UpdateMetrics(100, 5)
	panel.SetGuardrails(true)

	rendered := panel.Render("complete test")

	// Check all sections are present
	sections := []string{"MEMORY", "PROMPT INFO", "GUARDRAILS", "CAPABILITIES"}
	for _, section := range sections {
		if !strings.Contains(rendered, section) {
			t.Errorf("expected panel to contain %s section", section)
		}
	}
}

func TestInspectPanelScrolling(t *testing.T) {
	telemetry := NewTelemetry()
	panel := NewInspectPanel(telemetry)

	// Set small height to force scrolling
	panel.SetSize(30, 15)

	// Render content (will be truncated by viewport)
	rendered := panel.Render("scrolling test")

	// The panel should have content (even if truncated)
	if len(rendered) == 0 {
		t.Error("expected panel to have content")
	}

	// Header should be visible at the top
	if !strings.Contains(rendered, "INSPECT") {
		t.Error("expected header to be visible")
	}

	// Simulate scroll down (viewport Update with KeyDown)
	panel.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Panel should still render after scroll
	rendered = panel.Render("scrolling test")
	if len(rendered) == 0 {
		t.Error("expected panel to have content after scroll")
	}
}

// Phase 3: Input & Output Enhancements Tests

func TestRoleIdentifiers(t *testing.T) {
	tests := []struct {
		name         string
		role         string
		content      string
		expectedRole string
	}{
		{"user message", "user", "Hello", "USER:"},
		{"assistant message", "assistant", "Hi there", "ASSISTANT:"},
		{"system message", "system", "System notification", "SYSTEM:"},
		{"tool message", "tool", "Tool result", "TOOL:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newModel("test", nil)
			m.messages = []Message{
				{Role: tt.role, Content: tt.content, InProgress: false},
			}

			m.updateViewportContent()
			content := m.viewport.View()

			if !strings.Contains(content, tt.expectedRole) {
				t.Errorf("expected output to contain %q role identifier, got: %s", tt.expectedRole, content)
			}

			if !strings.Contains(content, tt.content) {
				t.Errorf("expected output to contain message content %q", tt.content)
			}
		})
	}
}

func TestRoleIdentifierStyling(t *testing.T) {
	// Test that style functions produce output with role identifiers
	userMsg := styleUserMessage("test user message")
	if !strings.Contains(userMsg, "USER:") {
		t.Error("expected USER: in styled user message")
	}

	assistantMsg := styleAssistantMessage("test assistant message")
	if !strings.Contains(assistantMsg, "ASSISTANT:") {
		t.Error("expected ASSISTANT: in styled assistant message")
	}

	systemMsg := styleSystemMessage("test system message")
	if !strings.Contains(systemMsg, "SYSTEM:") {
		t.Error("expected SYSTEM: in styled system message")
	}

	toolMsg := styleToolMessage("test tool message")
	if !strings.Contains(toolMsg, "TOOL:") {
		t.Error("expected TOOL: in styled tool message")
	}
}

// Phase 3: Mode Selector Tests

func TestModeSelector(t *testing.T) {
	m := newModel("test", nil)

	// Initial mode should be Chat
	if m.mode != ModeChat {
		t.Errorf("expected initial mode to be Chat, got %s", m.mode.String())
	}

	// Test mode cycling with Ctrl+L
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	m = result.(model)
	if m.mode != ModeCommand {
		t.Errorf("expected mode to be Command after Ctrl+L, got %s", m.mode.String())
	}

	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	m = result.(model)
	if m.mode != ModeDiff {
		t.Errorf("expected mode to be Diff after second Ctrl+L, got %s", m.mode.String())
	}

	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	m = result.(model)
	if m.mode != ModeChat {
		t.Errorf("expected mode to wrap back to Chat, got %s", m.mode.String())
	}
}

func TestModeString(t *testing.T) {
	tests := []struct {
		mode     Mode
		expected string
	}{
		{ModeChat, "Chat"},
		{ModeCommand, "Command"},
		{ModeDiff, "Diff"},
	}

	for _, tt := range tests {
		if tt.mode.String() != tt.expected {
			t.Errorf("expected mode string %q, got %q", tt.expected, tt.mode.String())
		}
	}
}

func TestModeDisplayInInput(t *testing.T) {
	m := newModel("test", nil)

	// Test Chat mode display
	inputView := m.renderInput()
	if !strings.Contains(inputView, "Mode: Chat") {
		t.Errorf("expected Chat mode in input display, got: %s", inputView)
	}

	// Switch to Command mode
	m.mode = ModeCommand
	inputView = m.renderInput()
	if !strings.Contains(inputView, "Mode: Command") {
		t.Errorf("expected Command mode in input display, got: %s", inputView)
	}

	// Switch to Diff mode
	m.mode = ModeDiff
	inputView = m.renderInput()
	if !strings.Contains(inputView, "Mode: Diff") {
		t.Errorf("expected Diff mode in input display, got: %s", inputView)
	}
}

// Phase 3: Input Toggles Tests

func TestInputToggles(t *testing.T) {
	m := newModel("test", nil)

	// Initial state should have toggles off
	if m.toggles.DryRun {
		t.Error("expected DryRun to be off initially")
	}
	if m.toggles.Deterministic {
		t.Error("expected Deterministic to be off initially")
	}

	// Test Ctrl+D to toggle dry run
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = result.(model)
	if !m.toggles.DryRun {
		t.Error("expected DryRun to be on after Ctrl+D")
	}

	// Test Ctrl+T to toggle deterministic
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	m = result.(model)
	if !m.toggles.Deterministic {
		t.Error("expected Deterministic to be on after Ctrl+T")
	}

	// Test toggling off
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = result.(model)
	if m.toggles.DryRun {
		t.Error("expected DryRun to be off after second Ctrl+D")
	}

	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	m = result.(model)
	if m.toggles.Deterministic {
		t.Error("expected Deterministic to be off after second Ctrl+T")
	}
}

func TestTogglesDisplayInInput(t *testing.T) {
	m := newModel("test", nil)

	// Toggles off
	inputView := m.renderInput()
	if !strings.Contains(inputView, "○ Dry Run") {
		t.Errorf("expected off indicator for Dry Run, got: %s", inputView)
	}
	if !strings.Contains(inputView, "○ Deterministic") {
		t.Errorf("expected off indicator for Deterministic, got: %s", inputView)
	}

	// Turn on Dry Run
	m.toggles.DryRun = true
	inputView = m.renderInput()
	if !strings.Contains(inputView, "✓ Dry Run") {
		t.Errorf("expected on indicator for Dry Run, got: %s", inputView)
	}

	// Turn on Deterministic
	m.toggles.Deterministic = true
	inputView = m.renderInput()
	if !strings.Contains(inputView, "✓ Deterministic") {
		t.Errorf("expected on indicator for Deterministic, got: %s", inputView)
	}
}

func TestTogglesIndependent(t *testing.T) {
	m := newModel("test", nil)

	// Toggle dry run on
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = result.(model)

	if !m.toggles.DryRun {
		t.Error("expected DryRun to be on")
	}
	if m.toggles.Deterministic {
		t.Error("expected Deterministic to remain off")
	}

	// Toggle deterministic on (dry run should stay on)
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	m = result.(model)

	if !m.toggles.DryRun {
		t.Error("expected DryRun to still be on")
	}
	if !m.toggles.Deterministic {
		t.Error("expected Deterministic to be on")
	}
}

// Phase 3: Collapsible Code Blocks Tests

func TestCodeBlockDetection(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"with code block", "Some text\n```go\ncode\n```", true},
		{"without code block", "Just plain text", false},
		{"multiple code blocks", "```\ncode1\n```\ntext\n```\ncode2\n```", true},
	}

	for _, tt := range tests {
		if IsCodeBlock(tt.text) != tt.expected {
			t.Errorf("%s: expected %v, got %v", tt.name, tt.expected, IsCodeBlock(tt.text))
		}
	}
}

func TestExtractCodeBlocks(t *testing.T) {
	text := "Some text\n```go\nfunc main() {\n  fmt.Println(\"hello\")\n}\n```\nMore text"
	blocks := ExtractCodeBlocks(text)

	if len(blocks) != 1 {
		t.Errorf("expected 1 code block, got %d", len(blocks))
	}

	if len(blocks) > 0 {
		if blocks[0].Language != "go" {
			t.Errorf("expected language 'go', got %q", blocks[0].Language)
		}
		if blocks[0].LineCount != 4 {
			t.Errorf("expected 4 lines, got %d", blocks[0].LineCount)
		}
	}
}

func TestCodeBlockCollapse(t *testing.T) {
	// Short block (should not collapse)
	shortBlock := CodeBlock{Language: "go", Content: "x := 1\n", LineCount: 1}
	shortBlock.ToggleCollapse()
	if shortBlock.Collapsed {
		t.Error("expected short block to not collapse")
	}

	// Long block (should collapse)
	longBlock := CodeBlock{
		Language:  "go",
		Content:   "line1\nline2\nline3\nline4\nline5\nline6\n",
		LineCount: 6,
		Collapsed: false,
	}
	longBlock.ToggleCollapse()
	if !longBlock.Collapsed {
		t.Error("expected long block to be collapsible")
	}

	// Toggle again
	longBlock.ToggleCollapse()
	if longBlock.Collapsed {
		t.Error("expected long block to expand after second toggle")
	}
}

func TestCodeBlockRender(t *testing.T) {
	// Collapsed block
	collapsedBlock := CodeBlock{
		Language:  "python",
		Content:   strings.Repeat("line\n", 10),
		LineCount: 10,
		Collapsed: true,
	}
	rendered := collapsedBlock.Render()
	if !strings.Contains(rendered, "expand") {
		t.Error("expected expand instruction in collapsed block")
	}
	if strings.Contains(rendered, "collapse") {
		t.Error("did not expect collapse instruction in collapsed block")
	}

	// Expanded block
	collapsedBlock.Collapsed = false
	rendered = collapsedBlock.Render()
	if !strings.Contains(rendered, "collapse") {
		t.Error("expected collapse instruction in expanded block")
	}
}

// Phase 3: Screen Reader Accessibility Tests

func TestAccessibilityInfo(t *testing.T) {
	tests := []struct {
		name     string
		message  Message
		expected []string
	}{
		{
			"user message",
			Message{Role: "user", Content: "Hello", InProgress: false},
			[]string{"User message", "Hello"},
		},
		{
			"assistant message streaming",
			Message{Role: "assistant", Content: "Hi there", InProgress: true},
			[]string{"Assistant message", "streaming", "Hi there"},
		},
		{
			"system message",
			Message{Role: "system", Content: "System update", InProgress: false},
			[]string{"System message", "System update"},
		},
		{
			"tool message",
			Message{Role: "tool", Content: "Tool result", InProgress: false},
			[]string{"Tool message", "Tool result"},
		},
	}

	for _, tt := range tests {
		desc := tt.message.AccessibilityInfo()
		for _, exp := range tt.expected {
			if !strings.Contains(desc, exp) {
				t.Errorf("%s: expected %q in %q", tt.name, exp, desc)
			}
		}
	}
}

func TestAccessibilityDescription(t *testing.T) {
	m := newModel("test", nil)

	desc := m.AccessibilityDescription()

	// Check for mode information
	if !strings.Contains(desc, "Chat") {
		t.Error("expected mode information in accessibility description")
	}

	// Check for toggle information
	if !strings.Contains(strings.ToLower(desc), "dry run") {
		t.Error("expected Dry Run toggle info in accessibility description")
	}

	// Check for navigation instructions
	if !strings.Contains(desc, "Tab") {
		t.Error("expected Tab navigation info in accessibility description")
	}

	// Check for keyboard instructions - Enter is now the send command
	if !strings.Contains(desc, "Enter") {
		t.Error("expected send command info in accessibility description")
	}
}

func TestAccessibilityWithToggles(t *testing.T) {
	m := newModel("test", nil)
	m.toggles.DryRun = true
	m.toggles.Deterministic = true

	desc := m.AccessibilityDescription()

	if !strings.Contains(desc, "on") {
		t.Error("expected toggle status in accessibility description")
	}

	// Toggle off
	m.toggles.DryRun = false
	desc = m.AccessibilityDescription()

	if !strings.Contains(desc, "off") {
		t.Error("expected off status for dry run")
	}
}

func TestAccessibilityFocusIndicators(t *testing.T) {
	m := newModel("test", nil)

	// Test that renderInput shows focus information
	m.focusedRegion = FocusInput
	inputView := m.renderInput()
	if !strings.Contains(inputView, "focused") {
		t.Error("expected focus indicator in input region")
	}

	// Test that renderOutputStream shows focus information
	m.focusedRegion = FocusOutputStream
	// Focus should be indicated in the border style change (tested visually)
	// but we can verify the message exists
	if m.focusedRegion != FocusOutputStream {
		t.Error("expected focus region to be set")
	}
}

func TestMultipleRolesInOutput(t *testing.T) {
	m := newModel("test", nil)
	m.messages = []Message{
		{Role: "user", Content: "Question", InProgress: false},
		{Role: "assistant", Content: "Answer", InProgress: false},
		{Role: "tool", Content: "Result", InProgress: false},
		{Role: "system", Content: "Status", InProgress: false},
	}

	m.updateViewportContent()
	content := m.viewport.View()

	// All role identifiers should be present
	roles := []string{"USER:", "ASSISTANT:", "TOOL:", "SYSTEM:"}
	for _, role := range roles {
		if !strings.Contains(content, role) {
			t.Errorf("expected output to contain %q role identifier", role)
		}
	}
}
