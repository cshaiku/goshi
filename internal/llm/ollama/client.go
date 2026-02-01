package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"grokgo/internal/llm"
)

type Client struct {
	baseURL string
	model   string
}

func New(model string) *Client {
	// Defensive defaulting
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
	messages []llm.Message,
) (llm.Stream, error) {

	reqBody := map[string]any{
		"model":    c.model,
		"messages": messages,
		"stream":   true,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
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

	return newStream(resp.Body), nil
}

