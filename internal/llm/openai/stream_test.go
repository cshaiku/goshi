package openai

import (
	"io"
	"strings"
	"testing"
)

// mockReadCloser wraps a string reader to make it an io.ReadCloser
type mockReadCloser struct {
	*strings.Reader
}

func (m *mockReadCloser) Close() error {
	return nil
}

func newMockReadCloser(s string) io.ReadCloser {
	return &mockReadCloser{Reader: strings.NewReader(s)}
}

func TestSSEStream_SingleChunk(t *testing.T) {
	sseData := `data: {"choices":[{"delta":{"content":"Hello"}}]}

data: [DONE]

`
	stream := newSSEStream(newMockReadCloser(sseData), nil, "gpt-4o")

	content, err := stream.Recv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if content != "Hello" {
		t.Errorf("expected 'Hello', got %q", content)
	}

	// Next recv should return EOF
	_, err = stream.Recv()
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestSSEStream_MultipleChunks(t *testing.T) {
	sseData := `data: {"choices":[{"delta":{"content":"Hello "}}]}

data: {"choices":[{"delta":{"content":"world"}}]}

data: {"choices":[{"delta":{"content":"!"}}]}

data: [DONE]

`
	stream := newSSEStream(newMockReadCloser(sseData), nil, "gpt-4o")

	// First recv - accumulates until buffer hits threshold or finishes
	content1, err := stream.Recv()
	if err != nil {
		t.Fatalf("unexpected error on first recv: %v", err)
	}

	// Should eventually get all content
	allContent := content1
	for {
		content, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		allContent += content
	}

	if allContent != "Hello world!" {
		t.Errorf("expected 'Hello world!', got %q", allContent)
	}
}

func TestSSEStream_EmptyContent(t *testing.T) {
	sseData := `data: {"choices":[{"delta":{"content":""}}]}

data: [DONE]

`
	stream := newSSEStream(newMockReadCloser(sseData), nil, "gpt-4o")

	// Should get EOF without content
	_, err := stream.Recv()
	if err != io.EOF {
		t.Errorf("expected EOF for empty content, got %v", err)
	}
}

func TestSSEStream_FinishReason(t *testing.T) {
	// Content and finish_reason typically come in separate chunks
	sseData := `data: {"choices":[{"delta":{"content":"Done"}}]}

data: {"choices":[{"delta":{},"finish_reason":"stop"}]}

`
	stream := newSSEStream(newMockReadCloser(sseData), nil, "gpt-4o")

	content, err := stream.Recv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if content != "Done" {
		t.Errorf("expected 'Done', got %q", content)
	}

	// Should be EOF on next call after finish_reason
	_, err = stream.Recv()
	if err != io.EOF {
		t.Errorf("expected EOF after finish_reason, got %v", err)
	}
}

func TestSSEStream_MalformedJSON(t *testing.T) {
	sseData := `data: not valid json

data: {"choices":[{"delta":{"content":"Hello"}}]}

data: [DONE]

`
	stream := newSSEStream(newMockReadCloser(sseData), nil, "gpt-4o")

	// Should skip malformed chunk and get valid content
	content := ""
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		content += chunk
	}

	if content != "Hello" {
		t.Errorf("expected 'Hello', got %q", content)
	}
}

func TestSSEStream_NoChoices(t *testing.T) {
	sseData := `data: {"choices":[]}

data: {"choices":[{"delta":{"content":"Hello"}}]}

data: [DONE]

`
	stream := newSSEStream(newMockReadCloser(sseData), nil, "gpt-4o")

	// Should skip empty choices and get valid content
	content := ""
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		content += chunk
	}

	if content != "Hello" {
		t.Errorf("expected 'Hello', got %q", content)
	}
}

func TestSSEStream_UsageData(t *testing.T) {
	// OpenAI sends usage in the final chunk
	sseData := `data: {"choices":[{"delta":{"content":"Hello"}}]}

data: {"choices":[{"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}

`
	costTracker := NewCostTracker("gpt-4o-mini", 0, 0)
	stream := newSSEStream(newMockReadCloser(sseData), costTracker, "gpt-4o-mini")

	// Read all content
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Check cost tracker recorded usage
	summary := costTracker.GetSummary()
	if summary.TotalPromptTokens != 10 {
		t.Errorf("expected 10 prompt tokens, got %d", summary.TotalPromptTokens)
	}
	if summary.TotalCompletionTokens != 5 {
		t.Errorf("expected 5 completion tokens, got %d", summary.TotalCompletionTokens)
	}
	if summary.RequestCount != 1 {
		t.Errorf("expected 1 request, got %d", summary.RequestCount)
	}
}

func TestSSEStream_NoCostTracker(t *testing.T) {
	sseData := `data: {"choices":[{"delta":{"content":"Hello"}}]}

data: [DONE]

`
	// No cost tracker - should still work
	stream := newSSEStream(newMockReadCloser(sseData), nil, "gpt-4o")

	content, err := stream.Recv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if content != "Hello" {
		t.Errorf("expected 'Hello', got %q", content)
	}
}

func TestSSEStream_LargeContent(t *testing.T) {
	// Build large content that exceeds buffer threshold (50 chars)
	largeContent := strings.Repeat("A", 100)

	sseData := `data: {"choices":[{"delta":{"content":"` + largeContent + `"}}]}

data: [DONE]

`
	stream := newSSEStream(newMockReadCloser(sseData), nil, "gpt-4o")

	// Should receive content in chunks due to buffer threshold
	allContent := ""
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		allContent += chunk
	}

	if allContent != largeContent {
		t.Errorf("expected %d chars, got %d", len(largeContent), len(allContent))
	}
}

func TestSSEStream_Close(t *testing.T) {
	sseData := `data: {"choices":[{"delta":{"content":"Hello"}}]}

`
	stream := newSSEStream(newMockReadCloser(sseData), nil, "gpt-4o")

	err := stream.Close()
	if err != nil {
		t.Errorf("unexpected close error: %v", err)
	}
}

func TestSSEStream_EmptyLines(t *testing.T) {
	sseData := `

data: {"choices":[{"delta":{"content":"Hello"}}]}


data: [DONE]

`
	stream := newSSEStream(newMockReadCloser(sseData), nil, "gpt-4o")

	// Should skip empty lines
	content := ""
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		content += chunk
	}

	if content != "Hello" {
		t.Errorf("expected 'Hello', got %q", content)
	}
}

func TestSSEStream_MultipleDONEFormats(t *testing.T) {
	// Test both [DONE] formats
	sseData := `data: {"choices":[{"delta":{"content":"Hello"}}]}

data: [DONE]

`
	stream := newSSEStream(newMockReadCloser(sseData), nil, "gpt-4o")

	content := ""
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		content += chunk
	}

	if content != "Hello" {
		t.Errorf("expected 'Hello', got %q", content)
	}
}

func TestSSEStream_EarlyEOF(t *testing.T) {
	// Stream ends without [DONE]
	sseData := `data: {"choices":[{"delta":{"content":"Hello"}}]}

`
	stream := newSSEStream(newMockReadCloser(sseData), nil, "gpt-4o")

	content := ""
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		content += chunk
	}

	// Should still get the content
	if content != "Hello" {
		t.Errorf("expected 'Hello', got %q", content)
	}
}

func TestUsageData_Integration(t *testing.T) {
	usage := &UsageData{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	if usage.PromptTokens != 100 {
		t.Errorf("expected 100 prompt tokens, got %d", usage.PromptTokens)
	}
	if usage.CompletionTokens != 50 {
		t.Errorf("expected 50 completion tokens, got %d", usage.CompletionTokens)
	}
	if usage.TotalTokens != 150 {
		t.Errorf("expected 150 total tokens, got %d", usage.TotalTokens)
	}
}
