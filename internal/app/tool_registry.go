package app

import (
	"encoding/json"
	"fmt"
	"sync"
)

// JSONSchema represents a JSON Schema for input validation
type JSONSchema struct {
	Type                 string                `json:"type"`
	Description          string                `json:"description,omitempty"`
	Properties           map[string]JSONSchema `json:"properties,omitempty"`
	Required             []string              `json:"required,omitempty"`
	AdditionalProperties bool                  `json:"additionalProperties"`
	Pattern              string                `json:"pattern,omitempty"`
}

// ToolDefinition describes a tool that the LLM can invoke
type ToolDefinition struct {
	ID                 string     `json:"id"`                   // "fs.read", "fs.write", "fs.list"
	Name               string     `json:"name"`                 // Human-readable name
	Description        string     `json:"description"`          // What the tool does
	RequiredPermission Capability `json:"-"`                    // Permission required to use
	Schema             JSONSchema `json:"inputSchema"`          // Input validation schema
	MaxRetries         int        `json:"maxRetries,omitempty"` // Default 0 (no retries)
}

// ToolRegistry is a centralized registry of all available tools
type ToolRegistry struct {
	tools map[string]ToolDefinition
	mu    sync.RWMutex
}

// NewToolRegistry creates an empty tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]ToolDefinition),
	}
}

// Register adds or updates a tool definition in the registry
func (r *ToolRegistry) Register(def ToolDefinition) error {
	if def.ID == "" {
		return fmt.Errorf("tool definition must have an ID")
	}
	if def.Name == "" {
		return fmt.Errorf("tool definition must have a name")
	}
	if def.Description == "" {
		return fmt.Errorf("tool definition must have a description")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[def.ID] = def
	return nil
}

// Get retrieves a tool definition by ID
// Returns the tool definition and a boolean indicating if it was found
func (r *ToolRegistry) Get(id string) (ToolDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[id]
	return tool, ok
}

// All returns all registered tool definitions
func (r *ToolRegistry) All() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}

// ValidateCall validates that the provided arguments match the tool's schema
func (r *ToolRegistry) ValidateCall(id string, args map[string]any) error {
	toolDef, ok := r.Get(id)
	if !ok {
		return fmt.Errorf("unknown tool: %s", id)
	}

	// Check required fields
	for _, field := range toolDef.Schema.Required {
		if _, ok := args[field]; !ok {
			return fmt.Errorf("missing required argument: %s", field)
		}
	}

	// Check that no extra fields are provided (if additionalProperties is false)
	if !toolDef.Schema.AdditionalProperties {
		for arg := range args {
			_, ok := toolDef.Schema.Properties[arg]
			if !ok {
				return fmt.Errorf("unexpected argument: %s", arg)
			}
		}
	}

	// Basic type validation for each argument
	for field, schema := range toolDef.Schema.Properties {
		if val, ok := args[field]; ok {
			if err := validateValue(val, schema); err != nil {
				return fmt.Errorf("invalid value for %s: %v", field, err)
			}
		}
	}

	return nil
}

func validateValue(val any, schema JSONSchema) error {
	switch schema.Type {
	case "string":
		if _, ok := val.(string); !ok {
			return fmt.Errorf("expected string, got %T", val)
		}
	case "number":
		switch val.(type) {
		case float64, int, int32, int64:
			// OK
		default:
			return fmt.Errorf("expected number, got %T", val)
		}
	case "boolean":
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", val)
		}
	case "array":
		if _, ok := val.([]any); !ok {
			return fmt.Errorf("expected array, got %T", val)
		}
	case "object":
		if _, ok := val.(map[string]any); !ok {
			return fmt.Errorf("expected object, got %T", val)
		}
	}
	return nil
}

// ToOpenAIFormat returns the tool definitions in OpenAI function calling format
func (r *ToolRegistry) ToOpenAIFormat() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]map[string]any, 0, len(r.tools))
	for _, tool := range r.tools {
		schemaJSON, _ := json.Marshal(tool.Schema)
		var schemaMap map[string]any
		json.Unmarshal(schemaJSON, &schemaMap)

		result = append(result, map[string]any{
			"name":        tool.ID,
			"description": tool.Description,
			"parameters":  schemaMap,
		})
	}
	return result
}
