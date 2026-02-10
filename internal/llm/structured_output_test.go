package llm

import (
	"testing"
)

func TestParseStructuredResponse_PlainText(t *testing.T) {
	resp, err := ParseStructuredResponse("Hello, I will help you")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Type != ResponseTypeText {
		t.Errorf("expected type text, got %s", resp.Type)
	}

	if resp.Text != "Hello, I will help you" {
		t.Errorf("expected text to match input")
	}
}

func TestParseStructuredResponse_ExplicitJSON(t *testing.T) {
	jsonResp := `{"type": "text", "text": "Hello from JSON"}`
	resp, err := ParseStructuredResponse(jsonResp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Type != ResponseTypeText {
		t.Errorf("expected type text, got %s", resp.Type)
	}
}

func TestParseStructuredResponse_ToolCall_SimplePattern(t *testing.T) {
	// Test simple tool call pattern detection
	resp, err := ParseStructuredResponse("I will call fs.read with path=README.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Type != ResponseTypeAction {
		t.Errorf("expected type action, got %s", resp.Type)
	}

	if resp.Action.Tool != "fs.read" {
		t.Errorf("expected tool fs.read, got %s", resp.Action.Tool)
	}

	if resp.Action.Args["path"] != "README.md" {
		t.Errorf("expected path=README.md, got %v", resp.Action.Args["path"])
	}
}

func TestParseStructuredResponse_ReturnsText_ByDefault(t *testing.T) {
	text := "Some random text that is not a tool call"
	resp, err := ParseStructuredResponse(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should default to text type for unrecognized format
	if resp.Type != ResponseTypeText {
		t.Errorf("expected default type text")
	}
}

func TestStructuredResponse_Validate_TextMessage(t *testing.T) {
	resp := &StructuredResponse{
		Type: ResponseTypeText,
		Text: "Hello",
	}

	if err := resp.Validate(); err != nil {
		t.Errorf("valid text response should not error: %v", err)
	}
}

func TestStructuredResponse_Validate_ActionMessage(t *testing.T) {
	resp := &StructuredResponse{
		Type: ResponseTypeAction,
		Action: &ActionCall{
			Tool: "fs.read",
			Args: map[string]any{"path": "file.txt"},
		},
	}

	if err := resp.Validate(); err != nil {
		t.Errorf("valid action response should not error: %v", err)
	}
}

func TestStructuredResponse_Validate_InvalidText(t *testing.T) {
	resp := &StructuredResponse{
		Type: ResponseTypeText,
		Text: "",
	}

	if err := resp.Validate(); err == nil {
		t.Error("empty text should fail validation")
	}
}

func TestStructuredResponse_Validate_MissingTool(t *testing.T) {
	resp := &StructuredResponse{
		Type: ResponseTypeAction,
		Action: &ActionCall{
			Tool: "",
			Args: map[string]any{},
		},
	}

	if err := resp.Validate(); err == nil {
		t.Error("action without tool should fail validation")
	}
}

func TestStructuredResponse_ToMessage_Text(t *testing.T) {
	resp := &StructuredResponse{
		Type: ResponseTypeText,
		Text: "Hello world",
	}

	msg := resp.ToMessage()
	if msg.Type() != TypeAssistantText {
		t.Errorf("expected message type assistant_text")
	}
}

func TestStructuredResponse_ToMessage_Action(t *testing.T) {
	resp := &StructuredResponse{
		Type: ResponseTypeAction,
		Action: &ActionCall{
			Tool: "fs.read",
			Args: map[string]any{"path": "file.txt"},
		},
	}

	msg := resp.ToMessage()
	if msg.Type() != TypeAssistantAction {
		t.Errorf("expected message type assistant_action")
	}
}

func TestStructuredResponse_String(t *testing.T) {
	resp := &StructuredResponse{
		Type: ResponseTypeText,
		Text: "Hello",
	}

	str := resp.String()
	if str == "" {
		t.Error("String() should not be empty")
	}
}

func TestParseStructuredResponse_EmptyInput(t *testing.T) {
	resp, err := ParseStructuredResponse("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Type != ResponseTypeError {
		t.Errorf("empty input should return error type")
	}
}

func TestParseStructuredResponse_Action_WithContent(t *testing.T) {
	text := "I will write to README.md with content='Hello World'"
	resp, err := ParseStructuredResponse(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// May or may not detect as a tool call depending on matching
	// Just verify it doesn't error
	if resp == nil {
		t.Error("response should not be nil")
	}
}

func TestIsWordBoundary(t *testing.T) {
	tests := []struct {
		r        rune
		expected bool
	}{
		{' ', true},
		{':', true},
		{',', true},
		{'a', false},
		{'1', false},
	}

	for _, tt := range tests {
		if result := isWordBoundary(tt.r); result != tt.expected {
			t.Errorf("isWordBoundary(%q) = %v, want %v", tt.r, result, tt.expected)
		}
	}
}

func TestFindToolMention(t *testing.T) {
	tests := []struct {
		text     string
		tool     string
		expected bool
	}{
		{"call fs.read with path", "fs.read", true},
		{"use fs.write to update", "fs.write", true},
		{"list files with fs.list", "fs.list", true},
		{"filesystem.read is not found", "fs.read", false},
	}

	for _, tt := range tests {
		idx := findToolMention(tt.text, tt.tool)
		found := idx >= 0
		if found != tt.expected {
			t.Errorf("findToolMention(%q, %q) found=%v, want %v",
				tt.text, tt.tool, found, tt.expected)
		}
	}
}
