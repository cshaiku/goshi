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

type Client struct {
	baseURL string
	model   string
}

// toolInstructions explicitly defines the JSON schema for local filesystem tools.
// This prevents the LLM from guessing file contents and forces it to use the toolchain.
const toolInstructions = `
# TOOL USE
If the user asks to list files, read a file, or write a file, you MUST NOT guess the contents.
Instead, you MUST output a single JSON object using one of the following formats:

- To list a directory: {"tool": "fs.list", "args": {"path": "."}}
- To read a file: {"tool": "fs.read", "args": {"path": "filename.txt"}}
- To write a file: {"tool": "fs.write", "args": {"path": "filename.txt", "content": "file contents here"}}

Do not provide conversational filler when triggering a tool. 
Output ONLY the JSON object.
`

func New(model string) *Client {
	if model == "" || model == "ollama" {
		model = "llama3"
	}
	return &Client{
		baseURL: "http://127.0.0.1:11434",
		model:   model,
	}
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
