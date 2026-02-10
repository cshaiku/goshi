package llm

// Message represents a single chat turn.
type Message struct {
  Role    string
  Content string
}

// Stream represents a streaming LLM response.
type Stream interface {
  Recv() (string, error)
  Close() error
}

type Chunk struct {
  Content string
}
