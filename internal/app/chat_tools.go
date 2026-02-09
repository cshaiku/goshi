package app

import (
	"encoding/json"
	"strings"

	"github.com/cshaiku/goshi/internal/llm"
)

func TryHandleToolCall(router *ToolRouter, text string) (*llm.Message, bool) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end <= start { return nil, false }

	jsonStr := text[start : end+1]
	var call struct {
		Tool string         `json:"tool"`
		Args map[string]any `json:"args"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &call); err != nil { return nil, false }
	if call.Tool == "" { return nil, false }

	result := router.Handle(ToolCall{Name: call.Tool, Args: call.Args})
	payload, _ := json.MarshalIndent(result, "", " ")
	return &llm.Message{Role: "system", Content: "Tool result:\n" + string(payload)}, true
}
