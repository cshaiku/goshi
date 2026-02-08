package app

import (
	"fmt"
)

type ToolAction interface {
	Run(args map[string]string) (string, error)
}

type ToolCall struct {
	Name string
	Args map[string]any
}

type ToolRouter struct {
	actions map[string]ToolAction
	caps    *Capabilities
}

func NewToolRouter(actions map[string]ToolAction, caps *Capabilities) *ToolRouter {
	return &ToolRouter{
		actions: actions,
		caps:    caps,
	}
}

// Handle executes a tool call requested by the LLM.
// NOTE: Step 1 — capabilities are NOT enforced yet.
func (r *ToolRouter) Handle(call ToolCall) any {
	action, ok := r.actions[call.Name]
	if !ok {
		return map[string]any{
			"error": fmt.Sprintf("tool not allowed: %s", call.Name),
		}
	}

	// Convert args to map[string]string (existing behavior assumption)
	args := map[string]string{}
	for k, v := range call.Args {
		if s, ok := v.(string); ok {
			args[k] = s
		}
	}

	result, err := action.Run(args)
	if err != nil {
		return map[string]any{
			"error": err.Error(),
		}
	}

	return map[string]any{
		"result": result,
	}
}