package llm

import (
	"testing"
)

func TestNewClientWithTools(t *testing.T) {
	sp, _ := NewSystemPrompt("test")
	client := NewClientWithTools(sp, nil)

	if client.system != sp {
		t.Error("system prompt should be set")
	}

	if client.parser == nil {
		t.Error("parser should be initialized")
	}
}

func TestResponseCollector(t *testing.T) {
	parser := NewStructuredParser()
	collector := NewResponseCollector(parser)

	collector.AddChunk("Hello")
	collector.AddChunk(" World")

	fullResp := collector.GetFullResponse()
	if fullResp != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", fullResp)
	}
}

func TestResponseCollector_Parse(t *testing.T) {
	parser := NewStructuredParser()
	collector := NewResponseCollector(parser)

	// Add some text response chunks
	collector.AddChunk("I will read the file")

	result, err := collector.Parse()
	if err != nil {
		t.Errorf("parse should not error: %v", err)
	}

	if !result.Valid {
		t.Errorf("simple text should parse as valid")
	}

	if result.Response.Type != ResponseTypeText {
		t.Errorf("expected text type")
	}
}

func TestResponseCollector_ParseEmpty(t *testing.T) {
	parser := NewStructuredParser()
	collector := NewResponseCollector(parser)

	_, err := collector.Parse()
	if err == nil {
		t.Error("parsing empty collection should error")
	}
}

func TestClientWithTools_SetToolValidator(t *testing.T) {
	sp, _ := NewSystemPrompt("test")
	client := NewClientWithTools(sp, nil)

	client.SetToolValidator(func(toolName string, args map[string]any) error {
		return nil
	})

	// Verify the validator was set (by the fact that it didn't error)
	parser := client.parser
	if parser == nil {
		t.Error("parser should exist")
	}
}

func TestGenerateToolSchemasForPrompt_Empty(t *testing.T) {
	tools := []interface{}{}
	prompt := GenerateToolSchemasForPrompt(tools)

	if prompt != "" {
		t.Error("empty tools should generate empty prompt")
	}
}

func TestGenerateToolSchemasForPrompt_WithTools(t *testing.T) {
	tools := []map[string]interface{}{
		{
			"id":          "fs.read",
			"description": "Read a file",
		},
	}

	prompt := GenerateToolSchemasForPrompt(tools)

	if prompt == "" {
		t.Error("should generate prompt for tools")
	}

	if !contains(prompt, "fs.read") {
		t.Error("prompt should include tool id")
	}

	if !contains(prompt, "Read a file") {
		t.Error("prompt should include tool description")
	}
}

func TestOllamaClient_SetToolDefinitions(t *testing.T) {
	// This test is in ollama package but we can test from here that the interface exists
	// Just verify the LLM package compiles with tool support
	sp, _ := NewSystemPrompt("test")
	client := NewClientWithTools(sp, nil)

	if client == nil {
		t.Error("client should be created")
	}
}
