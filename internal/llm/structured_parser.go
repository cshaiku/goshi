package llm

import "fmt"

// StructuredParser validates and parses LLM responses
// It uses a generic ToolValidator interface to avoid circular imports
type StructuredParser struct {
	validateToolCall func(toolName string, args map[string]any) error
}

// NewStructuredParser creates a basic parser
// Tools can be registered via SetValidator
func NewStructuredParser() *StructuredParser {
	return &StructuredParser{
		validateToolCall: func(toolName string, args map[string]any) error {
			return nil // Permissive by default
		},
	}
}

// SetToolValidator sets a function to validate tool calls
// This allows external packages to provide validation without circular imports
func (p *StructuredParser) SetToolValidator(fn func(toolName string, args map[string]any) error) {
	if fn != nil {
		p.validateToolCall = fn
	}
}

// ParseAndValidate parses raw LLM output and validates it
// Returns a validated StructuredResponse or an error with clear guidance
func (p *StructuredParser) ParseAndValidate(rawResponse string) (*StructuredResponse, error) {
	if rawResponse == "" {
		return nil, fmt.Errorf("empty response from LLM")
	}

	// Step 1: Parse into structured format
	response, err := ParseStructuredResponse(rawResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Step 2: Validate the response format
	if err := response.Validate(); err != nil {
		return nil, fmt.Errorf("invalid response structure: %w", err)
	}

	// Step 3: For action responses, validate via the validator
	if response.Type == ResponseTypeAction {
		if err := p.validateToolCall(response.Action.Tool, response.Action.Args); err != nil {
			return nil, fmt.Errorf("invalid tool call: %w", err)
		}
	}

	return response, nil
}

// ParseResult contains the result of a parse attempt with retry guidance
type ParseResult struct {
	Response   *StructuredResponse
	Valid      bool
	Error      string
	NeedsRetry bool
	Advice     string
}

// ParseWithRetryAdvice parses and provides guidance on whether to retry
func (p *StructuredParser) ParseWithRetryAdvice(rawResponse string) *ParseResult {
	response, err := p.ParseAndValidate(rawResponse)

	if err == nil {
		return &ParseResult{
			Response:   response,
			Valid:      true,
			NeedsRetry: false,
		}
	}

	// Determine if this is retryable (LLM error) or not (internal error)
	needsRetry := isLLMError(err.Error())
	advice := generateRetryAdvice(err.Error())

	return &ParseResult{
		Response:   response,
		Valid:      false,
		Error:      err.Error(),
		NeedsRetry: needsRetry,
		Advice:     advice,
	}
}

// isLLMError determines if an error is due to LLM output vs internal issues
func isLLMError(errStr string) bool {
	retryablePatterns := []string{
		"unknown tool",
		"invalid arguments",
		"invalid value",
	}

	for _, pattern := range retryablePatterns {
		if compareStrings(errStr, pattern) {
			return true
		}
	}

	return false
}

// generateRetryAdvice provides specific guidance for LLM retry
func generateRetryAdvice(errStr string) string {
	if compareStrings(errStr, "unknown tool") {
		return "The tool name is incorrect. Check the available tools and try again."
	}
	if compareStrings(errStr, "invalid") {
		return "Check the format of your request and try again."
	}
	return "Fix the issue shown above and try again."
}

// compareStrings checks if needle appears in haystack
func compareStrings(haystack, needle string) bool {
	return len(haystack) >= len(needle) && contains(haystack, needle)
}

// contains checks if a string contains a substring
func contains(text, substr string) bool {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// DeferToLLM returns a message asking the LLM to retry with guidance
func (result *ParseResult) DeferToLLM() string {
	return fmt.Sprintf(
		"[Parser Error]\n%s\n\n[Guidance]\n%s",
		result.Error,
		result.Advice,
	)
}
