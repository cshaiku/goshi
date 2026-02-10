package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cshaiku/goshi/internal/llm"
)

// toolInstructions defines the structured format for tool calling
const toolInstructions = `
## IMPORTANT: Tool Usage Instructions

When the user asks you to perform filesystem operations (list files, read files, write files),
you MUST call a tool. Do NOT attempt to guess or fabricate file contents.

### Response Format

When calling a tool, respond with ONLY a valid JSON object in one of these exact formats:

**To list directory contents:**
{"type": "action", "action": {"tool": "fs.list", "args": {"path": "."}}}

**To read a file:**
{"type": "action", "action": {"tool": "fs.read", "args": {"path": "README.md"}}}

**To write to a file:**
{"type": "action", "action": {"tool": "fs.write", "args": {"path": "file.txt", "content": "content here"}}}

**For planning/reasoning (NOT a tool call):**
{"type": "text", "text": "I will read the README file to understand the project"}

### Rules

1. If the user asks about file contents: ALWAYS use fs.read
2. If the user asks to list files: ALWAYS use fs.list
3. If the user asks to write/create/edit files: ALWAYS use fs.write
4. NEVER guess file contents - always use the tools
5. Respond only with JSON when using tools
6. Respond with natural text for planning and reasoning
`

type Client struct {
	baseURL  string
	model    string
	toolDefs string // Tool definitions to include in prompt
}

// NewClient creates an Ollama backend client
func New(model string) *Client {
	if model == "" || model == "ollama" {
		model = "llama3"
	}
	return &Client{
		baseURL:  "http://127.0.0.1:11434",
		model:    model,
		toolDefs: "",
	}
}

// SetToolDefinitions sets the tool definitions to include in the system prompt
// toolDefs should be a JSON string representing available tools
func (c *Client) SetToolDefinitions(toolDefs string) {
	c.toolDefs = toolDefs
}

func (c *Client) Stream(
	ctx context.Context,
	system string,
	messages []llm.Message,
) (llm.Stream, error) {
	reqMessages := make([]map[string]string, 0, len(messages)+1)

	// Combine the authoritative self-model with the tool-calling instructions [1, 2]
	combinedSystemPrompt := system + "\n" + toolInstructions

	reqMessages = append(reqMessages, map[string]string{
		"role":    "system",
		"content": combinedSystemPrompt,
	})

	for _, m := range messages {
		reqMessages = append(reqMessages, map[string]string{
			"role":    m.Role,
			"content": m.Content,
		})
	}

	reqBody := map[string]any{
		"model":    c.model,
		"messages": reqMessages,
		"stream":   true,
		"options": map[string]any{
			"temperature": 0.0, // Ensures deterministic tool calls rather than creative guesses [1]
		},
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/chat",
		bytes.NewReader(b),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"ollama /api/chat failed: %s: %s",
			resp.Status,
			string(body),
		)
	}

	// Use the authoritative stream implementation for response handling [3]
	return newStream(resp.Body), nil
}
