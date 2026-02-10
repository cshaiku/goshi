package app

// Tool definitions for all available tools
// These are registered once at startup and provide the single source of truth
// for what tools exist, their permissions, and their schemas

var (
	// FSReadTool reads a file from the repository
	FSReadTool = ToolDefinition{
		ID:                 "fs.read",
		Name:               "Read File",
		Description:        "Read the contents of a file from the repository. Path must be relative to the repository root.",
		RequiredPermission: CapFSRead,
		Schema: JSONSchema{
			Type:        "object",
			Description: "Arguments for reading a file",
			Properties: map[string]JSONSchema{
				"path": {
					Type:        "string",
					Description: "Relative path to the file within the repository",
				},
			},
			Required:             []string{"path"},
			AdditionalProperties: false,
		},
		MaxRetries: 0,
	}

	// FSWriteTool writes or creates a file in the repository
	FSWriteTool = ToolDefinition{
		ID:                 "fs.write",
		Name:               "Write File",
		Description:        "Write or create a file in the repository. Path must be relative to the repository root.",
		RequiredPermission: CapFSWrite,
		Schema: JSONSchema{
			Type:        "object",
			Description: "Arguments for writing a file",
			Properties: map[string]JSONSchema{
				"path": {
					Type:        "string",
					Description: "Relative path to the file within the repository",
				},
				"content": {
					Type:        "string",
					Description: "Content to write to the file",
				},
			},
			Required:             []string{"path", "content"},
			AdditionalProperties: false,
		},
		MaxRetries: 0,
	}

	// FSListTool lists files in a directory
	FSListTool = ToolDefinition{
		ID:                 "fs.list",
		Name:               "List Files",
		Description:        "List files and directories in a given path. Path must be relative to the repository root.",
		RequiredPermission: CapFSRead,
		Schema: JSONSchema{
			Type:        "object",
			Description: "Arguments for listing files",
			Properties: map[string]JSONSchema{
				"path": {
					Type:        "string",
					Description: "Relative path to the directory within the repository",
				},
			},
			Required:             []string{"path"},
			AdditionalProperties: false,
		},
		MaxRetries: 0,
	}
)

// NewToolRegistry creates a new registry with all default tools registered
func NewDefaultToolRegistry() *ToolRegistry {
	registry := NewToolRegistry()
	registry.Register(FSReadTool)
	registry.Register(FSWriteTool)
	registry.Register(FSListTool)
	return registry
}
