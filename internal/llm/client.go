package llm

import "context"

type Client struct {
	system  *SystemPrompt
	backend Backend
}

func NewClient(system *SystemPrompt, backend Backend) *Client {
	return &Client{
		system:  system,
		backend: backend,
	}
}

func (c *Client) Stream(
	ctx context.Context,
	messages []Message,
) (Stream, error) {
	return c.backend.Stream(ctx, c.system.Raw(), messages)
}
