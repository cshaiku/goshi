package openai

import (
	"encoding/json"

	"github.com/cshaiku/goshi/internal/app"
)

// ConvertToolsToOpenAIFormat converts Goshi tool definitions to OpenAI function calling format
func ConvertToolsToOpenAIFormat(tools []app.ToolDefinition) []map[string]any {
	if len(tools) == 0 {
		return nil
	}

	result := make([]map[string]any, 0, len(tools))

	for _, tool := range tools {
		// Convert JSON Schema to OpenAI parameters format
		parameters := convertSchemaToParameters(tool.Schema)

		functionDef := map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.ID,
				"description": tool.Description,
				"parameters":  parameters,
			},
		}

		result = append(result, functionDef)
	}

	return result
}

// convertSchemaToParameters converts Goshi JSON Schema to OpenAI parameters format
func convertSchemaToParameters(schema app.JSONSchema) map[string]any {
	params := map[string]any{
		"type":       schema.Type,
		"properties": make(map[string]any),
	}

	if len(schema.Required) > 0 {
		params["required"] = schema.Required
	}

	// Convert properties
	if schema.Properties != nil {
		props := make(map[string]any)
		for name, prop := range schema.Properties {
			propMap := map[string]any{
				"type": prop.Type,
			}
			if prop.Description != "" {
				propMap["description"] = prop.Description
			}
			if prop.Pattern != "" {
				propMap["pattern"] = prop.Pattern
			}
			props[name] = propMap
		}
		params["properties"] = props
	}

	params["additionalProperties"] = false

	return params
}

// ParseToolCallsFromResponse extracts tool calls from OpenAI response
// Returns empty slice if no tool calls found
func ParseToolCallsFromResponse(respData map[string]any) []ToolCall {
	choices, ok := respData["choices"].([]any)
	if !ok || len(choices) == 0 {
		return nil
	}

	choice, ok := choices[0].(map[string]any)
	if !ok {
		return nil
	}

	message, ok := choice["message"].(map[string]any)
	if !ok {
		return nil
	}

	toolCallsRaw, ok := message["tool_calls"].([]any)
	if !ok || len(toolCallsRaw) == 0 {
		return nil
	}

	result := make([]ToolCall, 0, len(toolCallsRaw))

	for _, tcRaw := range toolCallsRaw {
		tc, ok := tcRaw.(map[string]any)
		if !ok {
			continue
		}

		function, ok := tc["function"].(map[string]any)
		if !ok {
			continue
		}

		name, _ := function["name"].(string)
		argsJSON, _ := function["arguments"].(string)

		// Parse arguments JSON
		var args map[string]any
		if argsJSON != "" {
			// Unmarshal into map
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				// Skip malformed tool calls
				continue
			}
		}

		id, _ := tc["id"].(string)

		result = append(result, ToolCall{
			ID:   id,
			Name: name,
			Args: args,
		})
	}

	return result
}

// ToolCall represents a tool invocation from OpenAI
type ToolCall struct {
	ID   string
	Name string
	Args map[string]any
}
