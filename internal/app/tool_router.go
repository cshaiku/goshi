package app

import (
	"fmt"

	"github.com/cshaiku/goshi/internal/actions/runtime"
	"github.com/cshaiku/goshi/internal/audit"
)

type ToolCall struct {
	Name string
	Args map[string]any
}

type ToolRouter struct {
	dispatcher *runtime.Dispatcher
	registry   *ToolRegistry
	caps       *Capabilities
	auditLog   *audit.Logger
	auditCwd   string
}

func NewToolRouter(dispatcher *runtime.Dispatcher, caps *Capabilities) *ToolRouter {
	return &ToolRouter{
		dispatcher: dispatcher,
		registry:   NewDefaultToolRegistry(),
		caps:       caps,
	}
}

// NewToolRouterWithRegistry creates a tool router with a custom registry
func NewToolRouterWithRegistry(dispatcher *runtime.Dispatcher, registry *ToolRegistry, caps *Capabilities) *ToolRouter {
	return &ToolRouter{
		dispatcher: dispatcher,
		registry:   registry,
		caps:       caps,
	}
}

// SetAuditLogger configures the audit logger for tool calls.
func (r *ToolRouter) SetAuditLogger(logger *audit.Logger, cwd string) {
	r.auditLog = logger
	r.auditCwd = cwd
}

// Handle executes a tool call requested by the LLM.
// It validates the tool exists, validates the arguments against the schema,
// checks permissions, and then executes the tool via the dispatcher.
func (r *ToolRouter) Handle(call ToolCall) any {
	// Step 1: Look up tool definition
	toolDef, ok := r.registry.Get(call.Name)
	if !ok {
		r.logTool(call.Name, audit.StatusError, "unknown tool", call.Args)
		return map[string]any{
			"error": fmt.Sprintf("unknown tool: %s", call.Name),
		}
	}

	// Step 2: Validate call arguments against schema
	if err := r.registry.ValidateCall(call.Name, call.Args); err != nil {
		r.logTool(call.Name, audit.StatusError, fmt.Sprintf("invalid tool call: %v", err), call.Args)
		return map[string]any{
			"error": fmt.Sprintf("invalid tool call: %v", err),
		}
	}

	// Step 3: Check capability/permission enforcement
	if !r.caps.Has(toolDef.RequiredPermission) {
		r.logTool(call.Name, audit.StatusError, "permission denied", call.Args)
		return map[string]any{
			"error": fmt.Sprintf("permission denied for tool: %s", toolDef.ID),
		}
	}

	// Step 4: Execute the tool
	out, err := r.dispatcher.Dispatch(call.Name, runtime.ActionInput(call.Args))
	if err != nil {
		r.logTool(call.Name, audit.StatusError, err.Error(), call.Args)
		return map[string]any{
			"error": err.Error(),
		}
	}

	r.logTool(call.Name, audit.StatusOK, "ok", call.Args)

	return map[string]any{
		"result": out,
	}
}

func (r *ToolRouter) logTool(name string, status audit.EventStatus, message string, args map[string]any) {
	if r.auditLog == nil {
		return
	}
	r.auditLog.LogTool(name, status, message, args, r.auditCwd)
}

// ValidateToolCall validates that a tool call is valid without executing it.
// Returns an error if the tool is unknown, arguments are invalid, or permissions are missing.
func (r *ToolRouter) ValidateToolCall(toolName string, args map[string]any) error {
	// Step 1: Look up tool definition
	toolDef, ok := r.registry.Get(toolName)
	if !ok {
		return fmt.Errorf("unknown tool: %s", toolName)
	}

	// Step 2: Validate call arguments against schema
	if err := r.registry.ValidateCall(toolName, args); err != nil {
		return fmt.Errorf("invalid tool call: %w", err)
	}

	// Step 3: Check capability/permission enforcement
	if !r.caps.Has(toolDef.RequiredPermission) {
		return fmt.Errorf("permission denied for tool: %s", toolDef.ID)
	}

	return nil
}

// GetToolDefinitions returns all available tool definitions
// Useful for sending to LLM as function calling definitions
func (r *ToolRouter) GetToolDefinitions() []ToolDefinition {
	return r.registry.All()
}
