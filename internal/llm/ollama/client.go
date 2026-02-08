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

	// Inject system prompt ONCE, authoritatively
	reqMessages = append(reqMessages, map[string]string{
		"role":    "system",
		"content": system,
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

	// IMPORTANT: reuse the single authoritative stream implementation
	return newStream(resp.Body), nil
}
