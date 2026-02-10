package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ClientWithTools extends the basic Client with tool registry support
// and structured response parsing
type ClientWithTools struct {
	system   *SystemPrompt
	backend  Backend
	registry interface{} // Accepts tool registry (avoids circular import)
	parser   *StructuredParser
}

// NewClientWithTools creates a client with tool support
// registry should be *app.ToolRegistry but we use interface{} to avoid circular import
func NewClientWithTools(system *SystemPrompt, backend Backend) *ClientWithTools {
	return &ClientWithTools{
		system:   system,
		backend:  backend,
		registry: nil,
		parser:   NewStructuredParser(),
	}
}

// System returns the system prompt for this client
func (c *ClientWithTools) System() *SystemPrompt {
	return c.system
}

// Backend returns the LLM backend for this client
func (c *ClientWithTools) Backend() Backend {
	return c.backend
}

// SetToolRegistry attaches a tool registry to the client
// This allows the client to include tool schemas in prompts and validate responses
// registry should be *app.ToolRegistry
func (c *ClientWithTools) SetToolRegistry(registry interface{}) {
	c.registry = registry
}

// SetToolValidator sets the validation function for tool calls
func (c *ClientWithTools) SetToolValidator(validator func(toolName string, args map[string]any) error) {
	c.parser.SetToolValidator(validator)
}

// CollectStream reads the entire stream and returns the complete response
func (c *ClientWithTools) CollectStream(ctx context.Context, messages []Message) (string, error) {
	stream, err := c.backend.Stream(ctx, c.system.Raw(), messages)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	var fullResponse strings.Builder
	for {
		chunk, err := stream.Recv()
		if err != nil {
			break
		}
		fullResponse.WriteString(chunk)
	}

	return fullResponse.String(), nil
}

// StreamResponse streams the response while also collecting it for parsing
// Returns both the stream for partial output and the structured response
type StreamingResponse struct {
	Stream       Stream
	collector    strings.Builder
	parser       *StructuredParser
	isCollecting bool
}

// NewStreamingResponse creates a wrapper around a stream for collection and parsing
func NewStreamingResponse(stream Stream, parser *StructuredParser) *StreamingResponse {
	return &StreamingResponse{
		Stream:       stream,
		parser:       parser,
		isCollecting: true,
	}
}

// RecvWithCollection gets next chunk and collects it
func (sr *StreamingResponse) RecvWithCollection() (string, error) {
	chunk, err := sr.Stream.Recv()
	if sr.isCollecting {
		sr.collector.WriteString(chunk)
	}
	if err != nil {
		sr.isCollecting = false
	}
	return chunk, err
}

// GetCollectedResponse returns the full collected response
func (sr *StreamingResponse) GetCollectedResponse() string {
	return sr.collector.String()
}

// ParseCollectedResponse parses the collected response for structured output
func (sr *StreamingResponse) ParseCollectedResponse() (*ParseResult, error) {
	fullResponse := sr.GetCollectedResponse()
	if fullResponse == "" {
		return nil, fmt.Errorf("empty collected response")
	}

	return sr.parser.ParseWithRetryAdvice(fullResponse), nil
}

// ResponseCollector collects streamed chunks and provides full response for parsing
type ResponseCollector struct {
	chunks   []string
	fullText strings.Builder
	parser   *StructuredParser
}

// NewResponseCollector creates a response collector
func NewResponseCollector(parser *StructuredParser) *ResponseCollector {
	return &ResponseCollector{
		chunks: make([]string, 0),
		parser: parser,
	}
}

// AddChunk adds a streamed chunk to the collector
func (rc *ResponseCollector) AddChunk(chunk string) {
	rc.chunks = append(rc.chunks, chunk)
	rc.fullText.WriteString(chunk)
}

// GetFullResponse returns the complete collected response
func (rc *ResponseCollector) GetFullResponse() string {
	return rc.fullText.String()
}

// Parse returns a structured response from the collected chunks
func (rc *ResponseCollector) Parse() (*ParseResult, error) {
	fullResponse := rc.GetFullResponse()
	if fullResponse == "" {
		return nil, fmt.Errorf("no response collected")
	}

	return rc.parser.ParseWithRetryAdvice(fullResponse), nil
}

// ToolSchemaPrompt generates a prompt section describing available tools
func GenerateToolSchemasForPrompt(toolDefinitions interface{}) string {
	// toolDefinitions should be []app.ToolDefinition
	// but we receive it as interface{} to avoid circular imports

	data, err := json.Marshal(toolDefinitions)
	if err != nil {
		return ""
	}

	var tools []map[string]interface{}
	if err := json.Unmarshal(data, &tools); err != nil {
		return ""
	}

	if len(tools) == 0 {
		return ""
	}

	prompt := "\n## Available Tools\n"
	prompt += "You have access to the following tools. Call them when needed:\n\n"

	for _, tool := range tools {
		if id, ok := tool["id"].(string); ok {
			prompt += fmt.Sprintf("### %s\n", id)
		}
		if desc, ok := tool["description"].(string); ok {
			prompt += fmt.Sprintf("%s\n\n", desc)
		}
	}

	return prompt
}
