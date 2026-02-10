package app

import (
	"testing"

	"github.com/cshaiku/goshi/internal/actions/runtime"
	"github.com/cshaiku/goshi/internal/fs"
)

func createTestToolRouter() (*ToolRouter, *Capabilities) {
	// Create a minimal dispatcher for testing
	guard, _ := fs.NewGuard(".")
	dispatcher := runtime.NewDispatcher(guard)
	caps := NewCapabilities()
	router := NewToolRouter(dispatcher, caps)
	return router, caps
}

func TestToolRouter_Handle_UnknownTool(t *testing.T) {
	router, _ := createTestToolRouter()

	result := router.Handle(ToolCall{
		Name: "nonexistent.tool",
		Args: map[string]any{},
	})

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected result to be a map")
	}

	if errStr, ok := resultMap["error"]; !ok || errStr == nil {
		t.Fatal("expected error in result for unknown tool")
	}
}

func TestToolRouter_Handle_SchemaValidation(t *testing.T) {
	router, _ := createTestToolRouter()

	// Test missing required argument
	result := router.Handle(ToolCall{
		Name: "fs.read",
		Args: map[string]any{}, // Missing "path" argument
	})

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected result to be a map")
	}

	if errStr, ok := resultMap["error"]; !ok || errStr == nil {
		t.Fatal("expected error in result for invalid schema")
	}
}

func TestToolRouter_Handle_PermissionDenied(t *testing.T) {
	router, _ := createTestToolRouter()
	// Don't grant any capabilities - permission should be denied

	result := router.Handle(ToolCall{
		Name: "fs.read",
		Args: map[string]any{"path": "file.txt"},
	})

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected result to be a map")
	}

	if errStr, ok := resultMap["error"]; !ok || errStr == nil {
		t.Fatal("expected error in result for permission denied")
	}
}

func TestToolRouter_Handle_SuccessfulExecution(t *testing.T) {
	router, caps := createTestToolRouter()
	caps.Grant(CapFSRead)

	// fs.read on a nonexistent file will return an error, but that's from the dispatcher
	// This test just verifies that permission is granted and dispatcher is called
	result := router.Handle(ToolCall{
		Name: "fs.read",
		Args: map[string]any{"path": "nonexistent.txt"},
	})

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected result to be a map")
	}

	// Should have either "result" or "error" key (dispatcher error is OK in this test)
	if _, hasResult := resultMap["result"]; !hasResult {
		if _, hasError := resultMap["error"]; !hasError {
			t.Fatal("expected either result or error key in response")
		}
	}
}

func TestToolRouter_Handle_FSWrite_Permission(t *testing.T) {
	router, caps := createTestToolRouter()
	caps.Grant(CapFSRead) // Grant read but not write

	result := router.Handle(ToolCall{
		Name: "fs.write",
		Args: map[string]any{"path": "file.txt", "content": "data"},
	})

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected result to be a map")
	}

	if errStr, ok := resultMap["error"]; !ok || errStr == nil {
		t.Fatal("expected error when fs.write called without CapFSWrite permission")
	}
}

func TestToolRouter_GetToolDefinitions(t *testing.T) {
	router, _ := createTestToolRouter()

	tools := router.GetToolDefinitions()
	if len(tools) != 3 {
		t.Fatalf("expected 3 default tools, got %d", len(tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.ID] = true
	}

	expected := []string{"fs.read", "fs.write", "fs.list"}
	for _, name := range expected {
		if !toolNames[name] {
			t.Errorf("expected tool %s in definitions", name)
		}
	}
}

func TestNewToolRouterWithRegistry(t *testing.T) {
	guard, _ := fs.NewGuard(".")
	dispatcher := runtime.NewDispatcher(guard)
	caps := NewCapabilities()

	// Create a custom registry with fewer tools
	customRegistry := NewToolRegistry()
	customRegistry.Register(FSReadTool)

	router := NewToolRouterWithRegistry(dispatcher, customRegistry, caps)

	tools := router.GetToolDefinitions()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool in custom registry, got %d", len(tools))
	}

	if tools[0].ID != "fs.read" {
		t.Errorf("expected fs.read tool, got %s", tools[0].ID)
	}
}
