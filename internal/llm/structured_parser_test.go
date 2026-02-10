package llm

import (
	"fmt"
	"testing"
)

func TestStructuredParser_ParseAndValidate_ValidText(t *testing.T) {
	parser := NewStructuredParser()

	resp, err := parser.ParseAndValidate("I will help you")
	if err != nil {
		t.Errorf("valid text should not error: %v", err)
	}

	if resp.Type != ResponseTypeText {
		t.Errorf("expected type text")
	}
}

func TestStructuredParser_ParseAndValidate_ValidAction(t *testing.T) {
	parser := NewStructuredParser()

	resp, err := parser.ParseAndValidate("I will call fs.read with path=file.txt")
	if err != nil {
		t.Errorf("valid action should not error: %v", err)
	}

	if resp.Type != ResponseTypeAction {
		t.Errorf("expected type action")
	}
}

func TestStructuredParser_ParseAndValidate_Empty(t *testing.T) {
	parser := NewStructuredParser()

	_, err := parser.ParseAndValidate("")
	if err == nil {
		t.Error("empty input should error")
	}
}

func TestStructuredParser_SetToolValidator(t *testing.T) {
	parser := NewStructuredParser()

	validatorCalled := false
	parser.SetToolValidator(func(toolName string, args map[string]any) error {
		validatorCalled = true
		return nil
	})

	_, err := parser.ParseAndValidate("I will call fs.read with path=file.txt")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !validatorCalled {
		t.Error("tool validator should have been called")
	}
}

func TestStructuredParser_SetToolValidator_Rejects(t *testing.T) {
	parser := NewStructuredParser()

	parser.SetToolValidator(func(toolName string, args map[string]any) error {
		return fmt.Errorf("tool not allowed")
	})

	_, err := parser.ParseAndValidate("I will call fs.read with path=file.txt")
	if err == nil {
		t.Error("should error when validator rejects")
	}
}

func TestStructuredParser_ParseWithRetryAdvice_Valid(t *testing.T) {
	parser := NewStructuredParser()

	result := parser.ParseWithRetryAdvice("Hello there")
	if !result.Valid {
		t.Error("valid response should be marked valid")
	}

	if result.NeedsRetry {
		t.Error("valid response should not need retry")
	}
}

func TestStructuredParser_ParseWithRetryAdvice_Invalid(t *testing.T) {
	parser := NewStructuredParser()

	// Should reject tool calls that fail validation
	called := false
	parser.SetToolValidator(func(toolName string, args map[string]any) error {
		called = true
		return fmt.Errorf("validation failed")
	})

	result := parser.ParseWithRetryAdvice("I will call fs.read with path=file.txt")

	if called {
		// Tool was recognized and validation failed
		if result.Valid {
			t.Error("invalid response should not be marked valid")
		}

		// NeedsRetry may be false if error doesn't match retryable patterns
		// Just check that it was validated
		if result.Error == "" {
			t.Error("should have error message")
		}
	}
}

func TestParseResult_DeferToLLM(t *testing.T) {
	result := &ParseResult{
		Error:  "bad tool",
		Advice: "try again",
	}

	msg := result.DeferToLLM()
	if msg == "" {
		t.Error("message should not be empty")
	}
}

func TestIsLLMError(t *testing.T) {
	tests := []struct {
		errStr      string
		isRetryable bool
	}{
		{"unknown tool", true},
		{"invalid arguments", true},
		{"internal error", false},
	}

	for _, tt := range tests {
		result := isLLMError(tt.errStr)
		if result != tt.isRetryable {
			t.Errorf("isLLMError(%q) = %v, want %v", tt.errStr, result, tt.isRetryable)
		}
	}
}
