package app

import (
	"encoding/json"
	"strings"

	"github.com/cshaiku/goshi/internal/llm"
)

// tryHandleToolCall inspects assistant output and executes a tool call if present.
// It returns:
// - a new message to append to the conversation
// - a boolean indicating whether a tool was handled
func TryHandleToolCall(
	router *ToolRouter,
	text string,
) (*llm.Message, bool) {
	text = strings.TrimSpace(text)

	if !strings.HasPrefix(text, "{") || !strings.HasSuffix(text, "}") {
		return nil, false
	}

	var call struct {
		Tool string         `json:"tool"`
		Args map[string]any `json:"args"`
	}

	if err := json.Unmarshal([]byte(text), &call); err != nil {
		return nil, false
	}

	if call.Tool == "" {
		return nil, false
	}

	result := router.Handle(ToolCall{
		Name: call.Tool,
		Args: call.Args,
	})

	payload, _ := json.MarshalIndent(result, "", "  ")

	msg := llm.Message{
		Role:    "system",
		Content: "Tool result:\n" + string(payload),
	}

	return &msg, true
}
