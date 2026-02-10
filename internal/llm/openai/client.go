package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/cshaiku/goshi/internal/llm"
)

// toolInstructions defines the structured format for tool calling
// Similar to Ollama, but adapted for OpenAI's expectations
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

// Client implements the llm.Backend interface for OpenAI API
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	enableSSE  bool // Phase 2: Enable streaming via SSE
	maxRetries int  // Phase 2: Maximum retry attempts
}

// New creates an OpenAI backend client
// Loads API key from OPENAI_API_KEY environment variable
func New(model string) (*Client, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set\n\nTo use OpenAI:\n  1. Get an API key from https://platform.openai.com/api-keys\n  2. Set the environment variable:\n     export OPENAI_API_KEY='your-api-key-here'\n  3. Run goshi again")
	}

	// Default to gpt-4o-mini if not specified
	if model == "" || model == "openai" {
		model = "gpt-4o-mini"
	}

	return &Client{
		baseURL:    "https://api.openai.com/v1",
		apiKey:     apiKey,
		model:      model,
		enableSSE:  true, // Phase 2: Enable streaming
		maxRetries: 3,    // Phase 2: Default retry limit
	}, nil
}

// Stream sends a request to OpenAI and returns a streaming response
// Phase 2: Supports SSE streaming and retry logic with exponential backoff
func (c *Client) Stream(
	ctx context.Context,
	system string,
	messages []llm.Message,
) (llm.Stream, error) {
	var lastErr error

	// Retry loop with exponential backoff
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate backoff for retries
			backoff := CalculateBackoff(attempt-1, time.Second, 60*time.Second)
			fmt.Fprintf(os.Stderr, "[OpenAI] Retry attempt %d/%d after %v\n", attempt, c.maxRetries, backoff)

			select {
			case <-time.After(backoff):
				// Continue with retry
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		stream, err := c.doStream(ctx, system, messages)
		if err != nil {
			lastErr = err

			// Check if error is retryable
			if apiErr, ok := err.(*APIError); ok {
				if ShouldRetry(apiErr.StatusCode) && attempt < c.maxRetries {
					fmt.Fprintf(os.Stderr, "[OpenAI] Retryable error (%d): %s\n", apiErr.StatusCode, apiErr.Message)
					continue
				}
			}

			// Non-retryable error or out of retries
			return nil, err
		}

		// Success
		return stream, nil
	}

	return nil, fmt.Errorf("OpenAI request failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

// doStream performs the actual API request
func (c *Client) doStream(
	ctx context.Context,
	system string,
	messages []llm.Message,
) (llm.Stream, error) {
	// Build request messages array
	reqMessages := make([]map[string]string, 0, len(messages)+1)

	// Combine the authoritative self-model with the tool-calling instructions
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

	// Build request body
	reqBody := map[string]any{
		"model":       c.model,
		"messages":    reqMessages,
		"stream":      c.enableSSE, // Phase 2: Use SSE streaming
		"temperature": 0.0,         // Deterministic tool calls per Goshi design
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/chat/completions",
		bytes.NewReader(b),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API request failed: %w\n\nPossible causes:\n  - Network connectivity issues\n  - OpenAI API is down\n  - Firewall blocking https://api.openai.com", err)
	}

	// Handle HTTP errors
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, HandleHTTPError(resp, body)
	}

	// Phase 2: Return SSE stream if enabled
	if c.enableSSE {
		return newSSEStream(resp.Body), nil
	}

	// Fallback: Parse non-streaming response
	var respData struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w\nResponse: %s", err, string(body))
	}

	if len(respData.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned from OpenAI")
	}

	content := respData.Choices[0].Message.Content

	// Log token usage for visibility
	fmt.Fprintf(os.Stderr, "[OpenAI] Tokens - prompt: %d, completion: %d, total: %d (model: %s)\n",
		respData.Usage.PromptTokens,
		respData.Usage.CompletionTokens,
		respData.Usage.TotalTokens,
		c.model,
	)

	// Return a simple stream that returns the complete content once
	return &simpleStream{content: content, done: false}, nil
}

// simpleStream implements llm.Stream for non-streaming responses
// This is a Phase 1 implementation; Phase 2 will add true streaming
type simpleStream struct {
	content string
	done    bool
}

func (s *simpleStream) Recv() (string, error) {
	if s.done {
		return "", io.EOF
	}
	s.done = true
	return s.content, nil
}

func (s *simpleStream) Close() error {
	return nil
}
