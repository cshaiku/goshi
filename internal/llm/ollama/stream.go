package ollama

import (
	"bufio"
	"encoding/json"
	"io"
)

type stream struct {
	scanner *bufio.Scanner
	closer  io.Closer
	done    bool
}

func newStream(r io.ReadCloser) *stream {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	return &stream{
		scanner: scanner,
		closer:  r,
	}
}

func (s *stream) Recv() (string, error) {
	if s.done {
		return "", io.EOF
	}

	if !s.scanner.Scan() {
		return "", io.EOF
	}

	var chunk struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Done bool `json:"done"`
	}

	if err := json.Unmarshal(s.scanner.Bytes(), &chunk); err != nil {
		return "", err
	}

	if chunk.Done {
		s.done = true
		return "", io.EOF
	}

	return chunk.Message.Content, nil
}

func (s *stream) Close() error {
	return s.closer.Close()
}
