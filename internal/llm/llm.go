package llm

import "context"

type Message struct {
	Role    string
	Content string
}

type Stream interface {
	Next() bool
	Content() string
	Err() error
	Close() error
}

type ChatClient interface {
	Stream(ctx context.Context, messages []Message) (Stream, error)
}
