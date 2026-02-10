package openai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// sseStream implements llm.Stream for OpenAI's Server-Sent Events format
// Phase 3: Integrates cost tracking
type sseStream struct {
	reader      *bufio.Reader
	closer      io.ReadCloser
	buffer      strings.Builder
	done        bool
	lastErr     error
	costTracker *CostTracker // Phase 3: Track costs
	model       string       // Phase 3: Model for cost calculation
	usageData   *UsageData   // Phase 3: Accumulated usage stats
}

// UsageData tracks token usage from streaming responses
type UsageData struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// newSSEStream creates a streaming reader for OpenAI SSE responses
// Phase 3: Accepts cost tracker for usage tracking
func newSSEStream(body io.ReadCloser, costTracker *CostTracker, model string) *sseStream {
	return &sseStream{
		reader:      bufio.NewReader(body),
		closer:      body,
		done:        false,
		costTracker: costTracker,
		model:       model,
		usageData:   &UsageData{},
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
				s.recordUsage() // Phase 3: Record final usage
				return content, nil
			}
			s.recordUsage() // Phase 3: Record final usage
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
			s.recordUsage() // Phase 3: Record final usage
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
				s.recordUsage() // Phase 3: Record final usage
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
				Usage *UsageData `json:"usage"` // Phase 3: Usage data (in final chunk)
			}

			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				// Skip malformed chunks
				continue
			}

			// Phase 3: Capture usage data if present (OpenAI sends it in final chunk)
			if chunk.Usage != nil {
				s.usageData = chunk.Usage
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			choice := chunk.Choices[0]

			// Check if stream finished
			if choice.FinishReason != nil {
				s.done = true
				s.recordUsage() // Phase 3: Record final usage
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

// recordUsage records token usage and costs (Phase 3)
func (s *sseStream) recordUsage() {
	if s.costTracker == nil || s.usageData == nil {
		return
	}

	// Only record if we have actual usage data
	if s.usageData.PromptTokens == 0 && s.usageData.CompletionTokens == 0 {
		return
	}

	cost, warning, err := s.costTracker.RecordUsage(
		s.usageData.PromptTokens,
		s.usageData.CompletionTokens,
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "[OpenAI] ❌ Cost limit exceeded: %v\n", err)
	}

	if warning != "" {
		fmt.Fprintf(os.Stderr, "%s\n", warning)
	}

	// Log usage summary
	fmt.Fprintf(os.Stderr, "[OpenAI] Request cost: $%.4f | Tokens: %d prompt + %d completion = %d total (model: %s)\n",
		cost,
		s.usageData.PromptTokens,
		s.usageData.CompletionTokens,
		s.usageData.TotalTokens,
		s.model,
	)
}

// Close cleans up the stream
func (s *sseStream) Close() error {
	s.recordUsage() // Phase 3: Ensure usage is recorded on close
	return s.closer.Close()
}
