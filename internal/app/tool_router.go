package app

import (
	"errors"
)

// ToolCall represents a request coming from the LLM.
type ToolCall struct {
	Name string
	Args map[string]any
}

// ToolResult is returned back to the LLM.
type ToolResult struct {
	Name   string
	Output map[string]any
	Error  string
}

var (
	ErrToolNotAllowed = errors.New("tool not allowed")
)

// ToolRouter mediates between the LLM and local capabilities.
type ToolRouter struct {
	actions *ActionService
}

// NewToolRouter creates a ToolRouter scoped to a filesystem root.
func NewToolRouter(root string) (*ToolRouter, error) {
	actions, err := NewActionService(root)
	if err != nil {
		return nil, err
	}

	return &ToolRouter{
		actions: actions,
	}, nil
}

// Handle executes an allowed tool call and returns a structured result.
// This function MUST remain side-effect safe.
func (r *ToolRouter) Handle(call ToolCall) ToolResult {
	// Hard allowlist — do not relax casually
	switch call.Name {
	case "fs.read", "fs.list", "fs.write":
		// allowed
	default:
		return ToolResult{
			Name:  call.Name,
			Error: ErrToolNotAllowed.Error(),
		}
	}

	out, err := r.actions.RunAction(call.Name, call.Args)
	if err != nil {
		return ToolResult{
			Name:  call.Name,
			Error: err.Error(),
		}
	}

	return ToolResult{
		Name:   call.Name,
		Output: out,
	}
}
