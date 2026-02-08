package llm

import "context"

type Backend interface {
	Stream(
		ctx context.Context,
		system string,
		messages []Message,
	) (Stream, error)
}
