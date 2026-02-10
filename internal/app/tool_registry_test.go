package app

import (
	"testing"
)

func TestToolRegistry_Register(t *testing.T) {
	registry := NewToolRegistry()

	// Test valid registration
	err := registry.Register(FSReadTool)
	if err != nil {
		t.Fatalf("expected no error registering valid tool, got %v", err)
	}

	// Test that tool is retrievable
	tool, ok := registry.Get("fs.read")
	if !ok {
		t.Fatal("expected to find registered tool fs.read")
	}
	if tool.Name != "Read File" {
		t.Errorf("expected name 'Read File', got %q", tool.Name)
	}

	// Test error on missing ID
	badTool := ToolDefinition{Name: "No ID", Description: "Bad"}
	err = registry.Register(badTool)
	if err == nil {
		t.Fatal("expected error when registering tool without ID")
	}

	// Test error on missing name
	badTool = ToolDefinition{ID: "bad", Description: "No name"}
	err = registry.Register(badTool)
	if err == nil {
		t.Fatal("expected error when registering tool without name")
	}

	// Test error on missing description
	badTool = ToolDefinition{ID: "bad", Name: "No description"}
	err = registry.Register(badTool)
	if err == nil {
		t.Fatal("expected error when registering tool without description")
	}
}

func TestToolRegistry_Get(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(FSReadTool)

	// Test found tool
	tool, ok := registry.Get("fs.read")
	if !ok {
		t.Fatal("expected to find fs.read")
	}
	if tool.ID != "fs.read" {
		t.Errorf("expected ID 'fs.read', got %q", tool.ID)
	}

	// Test not found tool
	_, ok = registry.Get("nonexistent")
	if ok {
		t.Fatal("expected not found for nonexistent tool")
	}
}

func TestToolRegistry_All(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(FSReadTool)
	registry.Register(FSWriteTool)
	registry.Register(FSListTool)

	tools := registry.All()
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}

	toolIDs := make(map[string]bool)
	for _, tool := range tools {
		toolIDs[tool.ID] = true
	}

	for _, id := range []string{"fs.read", "fs.write", "fs.list"} {
		if !toolIDs[id] {
			t.Errorf("expected to find tool %s", id)
		}
	}
}

func TestToolRegistry_ValidateCall_Success(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(FSReadTool)
	registry.Register(FSWriteTool)

	// Valid call with required argument
	err := registry.ValidateCall("fs.read", map[string]any{"path": "/some/file.txt"})
	if err != nil {
		t.Errorf("expected valid call to succeed, got error: %v", err)
	}

	// Valid fs.write call
	err = registry.ValidateCall("fs.write", map[string]any{
		"path":    "/some/file.txt",
		"content": "hello world",
	})
	if err != nil {
		t.Errorf("expected valid write call to succeed, got error: %v", err)
	}
}

func TestToolRegistry_ValidateCall_Failures(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(FSReadTool)
	registry.Register(FSWriteTool)

	tests := []struct {
		name   string
		toolID string
		args   map[string]any
		errMsg string
	}{
		{
			name:   "unknown tool",
			toolID: "fs.nonexistent",
			args:   map[string]any{},
			errMsg: "unknown tool",
		},
		{
			name:   "missing required argument",
			toolID: "fs.read",
			args:   map[string]any{},
			errMsg: "missing required argument: path",
		},
		{
			name:   "fs.write missing content",
			toolID: "fs.write",
			args:   map[string]any{"path": "file.txt"},
			errMsg: "missing required argument: content",
		},
		{
			name:   "extra fields not allowed",
			toolID: "fs.read",
			args:   map[string]any{"path": "file.txt", "extra": "field"},
			errMsg: "unexpected argument: extra",
		},
		{
			name:   "wrong type for argument",
			toolID: "fs.read",
			args:   map[string]any{"path": 123},
			errMsg: "invalid value for path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.ValidateCall(tt.toolID, tt.args)
			if err == nil {
				t.Fatalf("expected error with message containing %q", tt.errMsg)
			}
		})
	}
}

func TestDefaultToolRegistry(t *testing.T) {
	registry := NewDefaultToolRegistry()
	tools := registry.All()

	if len(tools) != 3 {
		t.Fatalf("expected 3 default tools, got %d", len(tools))
	}

	// Verify each tool has correct permission requirement
	fsRead, _ := registry.Get("fs.read")
	if fsRead.RequiredPermission != CapFSRead {
		t.Errorf("fs.read should require CapFSRead")
	}

	fsWrite, _ := registry.Get("fs.write")
	if fsWrite.RequiredPermission != CapFSWrite {
		t.Errorf("fs.write should require CapFSWrite")
	}

	fsList, _ := registry.Get("fs.list")
	if fsList.RequiredPermission != CapFSRead {
		t.Errorf("fs.list should require CapFSRead")
	}
}

func TestToolRegistry_ToOpenAIFormat(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(FSReadTool)

	tools := registry.ToOpenAIFormat()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool in OpenAI format, got %d", len(tools))
	}

	tool := tools[0]
	if name, ok := tool["name"]; !ok || name != "fs.read" {
		t.Errorf("expected name 'fs.read' in OpenAI format")
	}

	if desc, ok := tool["description"]; !ok || desc == "" {
		t.Errorf("expected description in OpenAI format")
	}

	if params, ok := tool["parameters"]; !ok || params == nil {
		t.Errorf("expected parameters in OpenAI format")
	}
}
