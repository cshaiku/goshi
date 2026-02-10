package openai

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

// sseStream implements llm.Stream for OpenAI's Server-Sent Events format
type sseStream struct {
	reader  *bufio.Reader
	closer  io.ReadCloser
	buffer  strings.Builder
	done    bool
	lastErr error
}

// newSSEStream creates a streaming reader for OpenAI SSE responses
func newSSEStream(body io.ReadCloser) *sseStream {
	return &sseStream{
		reader: bufio.NewReader(body),
		closer: body,
		done:   false,
	}
}

// Recv reads the next chunk from the SSE stream
func (s *sseStream) Recv() (string, error) {
	if s.done {
		if s.lastErr != nil {
			return "", s.lastErr
		}
		return "", io.EOF
	}

	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			s.done = true
			s.lastErr = err
			if err == io.EOF && s.buffer.Len() > 0 {
				// Return any buffered content before EOF
				content := s.buffer.String()
				s.buffer.Reset()
				return content, nil
			}
			return "", err
		}

		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check for stream end marker
		if line == "data: [DONE]" {
			s.done = true
			if s.buffer.Len() > 0 {
				content := s.buffer.String()
				s.buffer.Reset()
				return content, nil
			}
			return "", io.EOF
		}

		// Parse SSE data lines
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Skip event markers
			if data == "[DONE]" {
				s.done = true
				if s.buffer.Len() > 0 {
					content := s.buffer.String()
					s.buffer.Reset()
					return content, nil
				}
				return "", io.EOF
			}

			// Parse JSON chunk
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
					FinishReason *string `json:"finish_reason"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				// Skip malformed chunks
				continue
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			choice := chunk.Choices[0]

			// Check if stream finished
			if choice.FinishReason != nil {
				s.done = true
				if s.buffer.Len() > 0 {
					content := s.buffer.String()
					s.buffer.Reset()
					return content, nil
				}
				return "", io.EOF
			}

			// Accumulate delta content
			if choice.Delta.Content != "" {
				s.buffer.WriteString(choice.Delta.Content)

				// Return buffered content periodically for responsiveness
				// Wait until we have meaningful content
				if s.buffer.Len() > 50 {
					content := s.buffer.String()
					s.buffer.Reset()
					return content, nil
				}
			}
		}
	}
}

// Close cleans up the stream
func (s *sseStream) Close() error {
	return s.closer.Close()
}
