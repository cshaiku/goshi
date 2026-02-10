package llm

import (
	"encoding/json"
	"fmt"
)

// ResponseType defines the type of LLM response
type ResponseType string

const (
	ResponseTypeText   ResponseType = "text"   // Plain text response (planning/reasoning)
	ResponseTypeAction ResponseType = "action" // Tool call/action request
	ResponseTypeError  ResponseType = "error"  // Error or clarification
)

// StructuredResponse represents a parsed LLM response
// with a clear type discriminator for routing and handling
type StructuredResponse struct {
	Type    ResponseType `json:"type"`
	Text    string       `json:"text,omitempty"`   // For ResponseTypeText
	Action  *ActionCall  `json:"action,omitempty"` // For ResponseTypeAction
	Error   string       `json:"error,omitempty"`  // For ResponseTypeError
	RawText string       `json:"-"`                // Original unparsed response
}

// ActionCall represents a tool invocation
type ActionCall struct {
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
}

// ParseStructuredResponse attempts to parse LLM response into structured format
// It tries multiple parsing strategies:
// 1. JSON with explicit type field
// 2. Tool call pattern detection
// 3. Plain text (default)
func ParseStructuredResponse(rawResponse string) (*StructuredResponse, error) {
	if rawResponse == "" {
		return &StructuredResponse{
			Type:    ResponseTypeError,
			Error:   "empty response from LLM",
			RawText: rawResponse,
		}, nil
	}

	// Strategy 1: Try parsing as explicit JSON structure
	if jsonResp := tryParseAsJSON(rawResponse); jsonResp != nil {
		jsonResp.RawText = rawResponse
		return jsonResp, nil
	}

	// Strategy 2: Try extracting tool call (pattern: `tool:name arg1=val1 arg2=val2`)
	if actionResp := tryExtractToolCall(rawResponse); actionResp != nil {
		actionResp.RawText = rawResponse
		return actionResp, nil
	}

	// Strategy 3: Default to plain text response
	return &StructuredResponse{
		Type:    ResponseTypeText,
		Text:    rawResponse,
		RawText: rawResponse,
	}, nil
}

// tryParseAsJSON attempts to parse response as structured JSON
func tryParseAsJSON(rawResponse string) *StructuredResponse {
	var data map[string]any
	if err := json.Unmarshal([]byte(rawResponse), &data); err != nil {
		return nil
	}

	// Check for type field
	typeVal, ok := data["type"].(string)
	if !ok {
		return nil
	}

	resp := &StructuredResponse{}

	switch ResponseType(typeVal) {
	case ResponseTypeText:
		if text, ok := data["text"].(string); ok {
			resp.Type = ResponseTypeText
			resp.Text = text
			return resp
		}

	case ResponseTypeAction:
		if actionData, ok := data["action"].(map[string]any); ok {
			tool, ok := actionData["tool"].(string)
			if !ok {
				return nil
			}

			args, ok := actionData["args"].(map[string]any)
			if !ok {
				args = make(map[string]any)
			}

			resp.Type = ResponseTypeAction
			resp.Action = &ActionCall{
				Tool: tool,
				Args: args,
			}
			return resp
		}

	case ResponseTypeError:
		if errMsg, ok := data["error"].(string); ok {
			resp.Type = ResponseTypeError
			resp.Error = errMsg
			return resp
		}
	}

	return nil
}

// tryExtractToolCall attempts to extract a tool call from unstructured text
// Looks for patterns like: "I will call fs.read with path=file.txt"
// Or: "tool: fs.read path=file.txt"
// Returns nil if no tool call pattern is detected
func tryExtractToolCall(rawResponse string) *StructuredResponse {
	// Known tools to match against
	knownTools := []string{"fs.read", "fs.write", "fs.list"}

	// Simple pattern matching for tool names
	for _, tool := range knownTools {
		idx := findToolMention(rawResponse, tool)
		if idx >= 0 {
			// Found a tool mention, try to extract arguments
			args := extractToolArgs(rawResponse[idx:])
			if len(args) > 0 || tool == "fs.list" {
				return &StructuredResponse{
					Type: ResponseTypeAction,
					Action: &ActionCall{
						Tool: tool,
						Args: args,
					},
				}
			}
		}
	}

	return nil
}

// findToolMention finds the index of a tool mention in text
func findToolMention(text, tool string) int {
	// Look for exact match or partial match
	idx := -1
	for i := 0; i <= len(text)-len(tool); i++ {
		if text[i:i+len(tool)] == tool {
			// Check word boundaries
			validBefore := i == 0 || isWordBoundary(rune(text[i-1]))
			validAfter := i+len(tool) >= len(text) || isWordBoundary(rune(text[i+len(tool)]))
			if validBefore && validAfter {
				idx = i
				break
			}
		}
	}
	return idx
}

// isWordBoundary checks if a rune is a word boundary
func isWordBoundary(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t' || r == ':' || r == ',' || r == '.' || r == '(' || r == ')'
}

// extractToolArgs attempts to extract key=value arguments from tool call text
func extractToolArgs(text string) map[string]any {
	args := make(map[string]any)

	// Look for path=/something or similar patterns
	// Simple implementation: find key=value pairs
	findPattern := func(key string) string {
		// Look for key=value pattern
		searchStr := key + "="
		idx := 0
		for {
			idx = findInText(text[idx:], searchStr)
			if idx < 0 {
				break
			}
			idx += len(searchStr)

			// Extract value until whitespace or delimiter
			valueStart := idx
			valueEnd := valueStart
			for valueEnd < len(text) && text[valueEnd] != ' ' && text[valueEnd] != '\n' && text[valueEnd] != ',' {
				valueEnd++
			}

			if valueEnd > valueStart {
				value := text[valueStart:valueEnd]
				// Clean up quotes if present
				if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') {
					value = value[1 : len(value)-1]
				}
				return value
			}
			idx = valueEnd
		}
		return ""
	}

	// Extract common arguments
	if path := findPattern("path"); path != "" {
		args["path"] = path
	}
	if content := findPattern("content"); content != "" {
		args["content"] = content
	}

	return args
}

// findInText finds substring in text
func findInText(text, substr string) int {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Validate ensures the structured response is valid
func (r *StructuredResponse) Validate() error {
	if r == nil {
		return fmt.Errorf("response is nil")
	}

	switch r.Type {
	case ResponseTypeText:
		if r.Text == "" {
			return fmt.Errorf("text response cannot be empty")
		}

	case ResponseTypeAction:
		if r.Action == nil {
			return fmt.Errorf("action response missing action data")
		}
		if r.Action.Tool == "" {
			return fmt.Errorf("action must have a tool name")
		}
		if r.Action.Args == nil {
			r.Action.Args = make(map[string]any)
		}

	case ResponseTypeError:
		if r.Error == "" {
			return fmt.Errorf("error response cannot be empty")
		}

	default:
		return fmt.Errorf("unknown response type: %s", r.Type)
	}

	return nil
}

// ToMessage converts the structured response to an LLMMessage for the conversation history
func (r *StructuredResponse) ToMessage() LLMMessage {
	switch r.Type {
	case ResponseTypeText:
		return NewAssistantTextMessage(r.Text)

	case ResponseTypeAction:
		return NewAssistantActionMessage(r.Action.Tool, r.Action.Args)

	case ResponseTypeError:
		// Errors are returned as text in the conversation
		return NewAssistantTextMessage(fmt.Sprintf("[ERROR] %s", r.Error))

	default:
		return NewAssistantTextMessage(r.RawText)
	}
}

// String provides a human-readable string representation
func (r *StructuredResponse) String() string {
	switch r.Type {
	case ResponseTypeText:
		return fmt.Sprintf("TextResponse: %s", r.Text)
	case ResponseTypeAction:
		return fmt.Sprintf("ActionResponse: %s(%v)", r.Action.Tool, r.Action.Args)
	case ResponseTypeError:
		return fmt.Sprintf("ErrorResponse: %s", r.Error)
	default:
		return fmt.Sprintf("UnknownResponse: %s", r.RawText)
	}
}
