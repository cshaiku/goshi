package xai

import (
	"context"

	"github.com/ZaguanLabs/xai-sdk-go/xai"
	"github.com/ZaguanLabs/xai-sdk-go/xai/chat"

	"grokgo/internal/llm"
)

type Client struct {
	model  string
	client *xai.Client
}

func New(apiKey, model string) (*Client, error) {
	cfg := xai.DefaultConfig()
	cfg.APIKey = apiKey

	c, err := xai.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		model:  model,
		client: c,
	}, nil
}

func (c *Client) Stream(ctx context.Context, messages []llm.Message) (llm.Stream, error) {
	var chatMsgs []*chat.Message

	for _, m := range messages {
		switch m.Role {
		case "system":
			chatMsgs = append(chatMsgs, chat.System(chat.Text(m.Content)))
		case "user":
			chatMsgs = append(chatMsgs, chat.User(chat.Text(m.Content)))
		case "assistant":
			chatMsgs = append(chatMsgs, chat.Assistant(chat.Text(m.Content)))
		}
	}

	req := chat.NewRequest(
		c.model,
		chat.WithTemperature(0.7),
		chat.WithMaxTokens(2048),
		chat.WithMessages(chatMsgs...),
	)

	stream, err := req.Stream(ctx, c.client.Chat())
	if err != nil {
		return nil, err
	}

	return &streamAdapter{stream: stream}, nil
}

type streamAdapter struct {
	stream *chat.Stream
}

func (s *streamAdapter) Next() bool {
	return s.stream.Next()
}

func (s *streamAdapter) Content() string {
	return s.stream.Current().Content()
}

func (s *streamAdapter) Err() error {
	return s.stream.Err()
}

func (s *streamAdapter) Close() error {
	return s.stream.Close()
}
