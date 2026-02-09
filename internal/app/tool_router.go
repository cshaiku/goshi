package app

import (
	"github.com/cshaiku/goshi/internal/actions/runtime"
)

type ToolCall struct {
	Name string
	Args map[string]any
}

type ToolRouter struct {
	dispatcher *runtime.Dispatcher
	caps       *Capabilities
}

func NewToolRouter(dispatcher *runtime.Dispatcher, caps *Capabilities) *ToolRouter {
	return &ToolRouter{
		dispatcher: dispatcher,
		caps:       caps,
	}
}

// Handle executes a tool call requested by the LLM.
func (r *ToolRouter) Handle(call ToolCall) any {
	// --- STEP 2: CAPABILITY ENFORCEMENT ---
	switch call.Name {
	case "fs.read", "fs.list":
		if !r.caps.Has(CapFSRead) {
			return map[string]any{
				"error": "filesystem read access not granted for this session",
			}
		}
	}

	out, err := r.dispatcher.Dispatch(call.Name, runtime.ActionInput(call.Args))
	if err != nil {
		return map[string]any{
			"error": err.Error(),
		}
	}

	return map[string]any{
		"result": out,
	}
}
