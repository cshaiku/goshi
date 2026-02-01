package llm

import "context"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Stream interface {
	Recv() (string, error)
	Close() error
}

type Client interface {
	Stream(ctx context.Context, messages []Message) (Stream, error)
}

