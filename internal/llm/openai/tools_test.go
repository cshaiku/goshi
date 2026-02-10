package openai

import (
	"encoding/json"
	"testing"

	"github.com/cshaiku/goshi/internal/app"
)

func TestConvertToolsToOpenAIFormat_Empty(t *testing.T) {
	result := ConvertToolsToOpenAIFormat(nil)
	if result != nil {
		t.Error("expected nil for empty tools")
	}

	result = ConvertToolsToOpenAIFormat([]app.ToolDefinition{})
	if result != nil {
		t.Error("expected nil for empty tools slice")
	}
}

func TestConvertToolsToOpenAIFormat_SingleTool(t *testing.T) {
	tools := []app.ToolDefinition{
		{
			ID:          "fs.read",
			Description: "Read a file",
			Schema: app.JSONSchema{
				Type: "object",
				Properties: map[string]app.JSONSchema{
					"path": {
						Type:        "string",
						Description: "File path",
					},
				},
				Required: []string{"path"},
			},
		},
	}

	result := ConvertToolsToOpenAIFormat(tools)

	if len(result) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result))
	}

	tool := result[0]

	// Check type
	if tool["type"] != "function" {
		t.Errorf("expected type 'function', got %v", tool["type"])
	}

	// Check function definition
	function, ok := tool["function"].(map[string]any)
	if !ok {
		t.Fatal("function should be a map")
	}

	if function["name"] != "fs.read" {
		t.Errorf("expected name 'fs.read', got %v", function["name"])
	}

	if function["description"] != "Read a file" {
		t.Errorf("expected description 'Read a file', got %v", function["description"])
	}

	// Check parameters
	params, ok := function["parameters"].(map[string]any)
	if !ok {
		t.Fatal("parameters should be a map")
	}

	if params["type"] != "object" {
		t.Errorf("expected type 'object', got %v", params["type"])
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map")
	}

	pathProp, ok := props["path"].(map[string]any)
	if !ok {
		t.Fatal("path property should exist")
	}

	if pathProp["type"] != "string" {
		t.Errorf("expected path type 'string', got %v", pathProp["type"])
	}

	if pathProp["description"] != "File path" {
		t.Errorf("expected path description 'File path', got %v", pathProp["description"])
	}

	// Check required
	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("required should be a string slice")
	}

	if len(required) != 1 || required[0] != "path" {
		t.Errorf("expected required ['path'], got %v", required)
	}
}

func TestConvertToolsToOpenAIFormat_MultipleTools(t *testing.T) {
	tools := []app.ToolDefinition{
		{
			ID:          "fs.read",
			Description: "Read a file",
			Schema: app.JSONSchema{
				Type:       "object",
				Properties: map[string]app.JSONSchema{},
			},
		},
		{
			ID:          "fs.write",
			Description: "Write a file",
			Schema: app.JSONSchema{
				Type:       "object",
				Properties: map[string]app.JSONSchema{},
			},
		},
	}

	result := ConvertToolsToOpenAIFormat(tools)

	if len(result) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(result))
	}

	// Check first tool
	func1 := result[0]["function"].(map[string]any)
	if func1["name"] != "fs.read" {
		t.Errorf("expected first tool 'fs.read', got %v", func1["name"])
	}

	// Check second tool
	func2 := result[1]["function"].(map[string]any)
	if func2["name"] != "fs.write" {
		t.Errorf("expected second tool 'fs.write', got %v", func2["name"])
	}
}

func TestConvertToolsToOpenAIFormat_WithPattern(t *testing.T) {
	tools := []app.ToolDefinition{
		{
			ID:          "test.tool",
			Description: "Test tool",
			Schema: app.JSONSchema{
				Type: "object",
				Properties: map[string]app.JSONSchema{
					"email": {
						Type:        "string",
						Description: "Email address",
						Pattern:     "^[a-z]+@[a-z]+\\.[a-z]+$",
					},
				},
			},
		},
	}

	result := ConvertToolsToOpenAIFormat(tools)

	function := result[0]["function"].(map[string]any)
	params := function["parameters"].(map[string]any)
	props := params["properties"].(map[string]any)
	emailProp := props["email"].(map[string]any)

	if emailProp["pattern"] != "^[a-z]+@[a-z]+\\.[a-z]+$" {
		t.Errorf("expected pattern, got %v", emailProp["pattern"])
	}
}

func TestConvertToolsToOpenAIFormat_NoRequiredFields(t *testing.T) {
	tools := []app.ToolDefinition{
		{
			ID:          "test.tool",
			Description: "Test tool",
			Schema: app.JSONSchema{
				Type:       "object",
				Properties: map[string]app.JSONSchema{},
				Required:   []string{}, // Empty required
			},
		},
	}

	result := ConvertToolsToOpenAIFormat(tools)

	function := result[0]["function"].(map[string]any)
	params := function["parameters"].(map[string]any)

	// Empty required should not be included
	if _, hasRequired := params["required"]; hasRequired {
		required := params["required"].([]string)
		if len(required) > 0 {
			t.Error("empty required array should not be included")
		}
	}
}

func TestParseToolCallsFromResponse_Empty(t *testing.T) {
	respData := map[string]any{}
	result := ParseToolCallsFromResponse(respData)

	if result != nil {
		t.Error("expected nil for empty response")
	}
}

func TestParseToolCallsFromResponse_NoToolCalls(t *testing.T) {
	respData := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"content": "Hello",
				},
			},
		},
	}

	result := ParseToolCallsFromResponse(respData)

	if result != nil {
		t.Error("expected nil when no tool_calls present")
	}
}

func TestParseToolCallsFromResponse_SingleToolCall(t *testing.T) {
	argsJSON := `{"path": "/tmp/test.txt"}`

	respData := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"tool_calls": []any{
						map[string]any{
							"id": "call_123",
							"function": map[string]any{
								"name":      "fs.read",
								"arguments": argsJSON,
							},
						},
					},
				},
			},
		},
	}

	result := ParseToolCallsFromResponse(respData)

	if len(result) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result))
	}

	tc := result[0]

	if tc.ID != "call_123" {
		t.Errorf("expected ID 'call_123', got %q", tc.ID)
	}

	if tc.Name != "fs.read" {
		t.Errorf("expected name 'fs.read', got %q", tc.Name)
	}

	if tc.Args["path"] != "/tmp/test.txt" {
		t.Errorf("expected args path '/tmp/test.txt', got %v", tc.Args["path"])
	}
}

func TestParseToolCallsFromResponse_MultipleToolCalls(t *testing.T) {
	respData := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"tool_calls": []any{
						map[string]any{
							"id": "call_1",
							"function": map[string]any{
								"name":      "fs.read",
								"arguments": `{"path": "a.txt"}`,
							},
						},
						map[string]any{
							"id": "call_2",
							"function": map[string]any{
								"name":      "fs.write",
								"arguments": `{"path": "b.txt", "content": "test"}`,
							},
						},
					},
				},
			},
		},
	}

	result := ParseToolCallsFromResponse(respData)

	if len(result) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(result))
	}

	// Check first call
	if result[0].Name != "fs.read" {
		t.Errorf("expected first call 'fs.read', got %q", result[0].Name)
	}

	if result[0].Args["path"] != "a.txt" {
		t.Errorf("expected path 'a.txt', got %v", result[0].Args["path"])
	}

	// Check second call
	if result[1].Name != "fs.write" {
		t.Errorf("expected second call 'fs.write', got %q", result[1].Name)
	}

	if result[1].Args["content"] != "test" {
		t.Errorf("expected content 'test', got %v", result[1].Args["content"])
	}
}

func TestParseToolCallsFromResponse_MalformedJSON(t *testing.T) {
	respData := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"tool_calls": []any{
						map[string]any{
							"id": "call_bad",
							"function": map[string]any{
								"name":      "fs.read",
								"arguments": `{invalid json}`,
							},
						},
						map[string]any{
							"id": "call_good",
							"function": map[string]any{
								"name":      "fs.write",
								"arguments": `{"path": "ok.txt"}`,
							},
						},
					},
				},
			},
		},
	}

	result := ParseToolCallsFromResponse(respData)

	// Should skip malformed and return valid one
	if len(result) != 1 {
		t.Fatalf("expected 1 tool call (skipping malformed), got %d", len(result))
	}

	if result[0].Name != "fs.write" {
		t.Errorf("expected 'fs.write', got %q", result[0].Name)
	}
}

func TestParseToolCallsFromResponse_EmptyArguments(t *testing.T) {
	respData := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"tool_calls": []any{
						map[string]any{
							"id": "call_123",
							"function": map[string]any{
								"name":      "fs.list",
								"arguments": "",
							},
						},
					},
				},
			},
		},
	}

	result := ParseToolCallsFromResponse(respData)

	if len(result) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result))
	}

	if result[0].Args != nil {
		t.Errorf("expected nil args for empty arguments, got %v", result[0].Args)
	}
}

func TestToolCall_Structure(t *testing.T) {
	tc := ToolCall{
		ID:   "test_id",
		Name: "test_name",
		Args: map[string]any{"key": "value"},
	}

	if tc.ID != "test_id" {
		t.Errorf("expected ID 'test_id', got %q", tc.ID)
	}

	if tc.Name != "test_name" {
		t.Errorf("expected Name 'test_name', got %q", tc.Name)
	}

	if tc.Args["key"] != "value" {
		t.Errorf("expected Args key 'value', got %v", tc.Args["key"])
	}
}

func TestConvertToolsToOpenAIFormat_AdditionalProperties(t *testing.T) {
	tools := []app.ToolDefinition{
		{
			ID:          "test.tool",
			Description: "Test",
			Schema: app.JSONSchema{
				Type:       "object",
				Properties: map[string]app.JSONSchema{},
			},
		},
	}

	result := ConvertToolsToOpenAIFormat(tools)

	function := result[0]["function"].(map[string]any)
	params := function["parameters"].(map[string]any)

	// Should explicitly set additionalProperties to false
	if params["additionalProperties"] != false {
		t.Errorf("expected additionalProperties false, got %v", params["additionalProperties"])
	}
}

func TestConvertSchemaToParameters_ComplexSchema(t *testing.T) {
	schema := app.JSONSchema{
		Type: "object",
		Properties: map[string]app.JSONSchema{
			"name": {
				Type:        "string",
				Description: "User name",
			},
			"age": {
				Type:        "string",
				Description: "User age",
			},
			"email": {
				Type:        "string",
				Description: "Email address",
				Pattern:     "^.+@.+\\..+$",
			},
		},
		Required: []string{"name", "email"},
	}

	// This tests the helper function through ConvertToolsToOpenAIFormat
	tools := []app.ToolDefinition{{ID: "test", Description: "Test", Schema: schema}}
	result := ConvertToolsToOpenAIFormat(tools)

	function := result[0]["function"].(map[string]any)
	params := function["parameters"].(map[string]any)
	props := params["properties"].(map[string]any)

	// Check all properties converted
	if len(props) != 3 {
		t.Errorf("expected 3 properties, got %d", len(props))
	}

	// Check name property
	nameProp := props["name"].(map[string]any)
	if nameProp["type"] != "string" {
		t.Error("name should be string type")
	}

	// Check age property
	ageProp := props["age"].(map[string]any)
	if ageProp["type"] != "string" {
		t.Error("age should be string type")
	}

	// Check email with pattern
	emailProp := props["email"].(map[string]any)
	if emailProp["pattern"] != "^.+@.+\\..+$" {
		t.Error("email pattern not preserved")
	}

	// Check required array
	required := params["required"].([]string)
	if len(required) != 2 {
		t.Errorf("expected 2 required fields, got %d", len(required))
	}
}

func TestParseToolCallsFromResponse_ComplexArgs(t *testing.T) {
	// Test with nested JSON arguments
	argsJSON := `{
		"path": "/tmp/test.txt",
		"options": {
			"encoding": "utf-8",
			"create": true
		},
		"tags": ["important", "draft"]
	}`

	respData := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"tool_calls": []any{
						map[string]any{
							"id": "call_complex",
							"function": map[string]any{
								"name":      "fs.write",
								"arguments": argsJSON,
							},
						},
					},
				},
			},
		},
	}

	result := ParseToolCallsFromResponse(respData)

	if len(result) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result))
	}

	args := result[0].Args

	// Check path
	if args["path"] != "/tmp/test.txt" {
		t.Errorf("path not parsed correctly")
	}

	// Check nested options
	options, ok := args["options"].(map[string]any)
	if !ok {
		t.Fatal("options should be a map")
	}

	if options["encoding"] != "utf-8" {
		t.Error("nested encoding not parsed")
	}

	// Check array
	tags, ok := args["tags"].([]any)
	if !ok {
		t.Fatal("tags should be an array")
	}

	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

// Helper to parse JSON and check structure
func parseJSON(t *testing.T, data string) map[string]any {
	var result map[string]any
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	return result
}
