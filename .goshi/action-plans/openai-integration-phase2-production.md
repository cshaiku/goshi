# OpenAI Integration - Phase 2: Production Ready

**Status**: Not Started  
**Phase**: 2 of 3  
**Commit Descriptor**: `[openai-integration]`  
**Date**: 2026-02-10

## Objective
Enhance OpenAI backend with production-grade features: streaming responses, native tool calling, and comprehensive error handling.

## Dependencies
- Phase 1 complete (basic OpenAI client working)

## Implementation Tasks

### 1. Streaming Support (SSE)
**File**: `internal/llm/openai/stream.go`
- Implement Server-Sent Events (SSE) parser
- Handle `data:` lines with JSON deltas
- Accumulate content across stream chunks
- Handle `delta` field updates correctly
- Proper cleanup on stream close/error
- Implement `llm.Stream` interface matching Ollama

**File**: `internal/llm/openai/client.go`
- Update Stream() method to use streaming API (`stream: true`)
- Return SSE stream wrapper

### 2. Native Tool Calling (Function Calling)
**File**: `internal/llm/openai/tools.go`
- Convert Goshi tool definitions to OpenAI `tools` format
- Implement proper JSON schema mapping
- Handle `tool_choice` parameter ("auto", "required", specific tool)
- Parse `tool_calls` from response messages
- Support `parallel_tool_calls` parameter
- Accumulate tool calls across streaming chunks

**File**: `internal/llm/openai/client.go`
- Inject tools array in request payload
- Remove hardcoded tool instructions (use native OpenAI tools instead)
- Keep system prompt for self-model context only

### 3. Comprehensive Error Handling
**File**: `internal/llm/openai/errors.go`
- 401 Unauthorized: "Invalid API key" with setup guidance
- 429 Rate Limited: Exponential backoff (configurable)
- 500 Server Error: Retry with backoff
- 503 Service Unavailable: Retry or fallback guidance
- Timeout errors: Use `request_timeout` from config
- Network errors: Clear error messages with troubleshooting hints

**File**: `internal/llm/openai/client.go`
- Integrate error handling in request/response cycle
- Log errors with appropriate severity
- Provide actionable user feedback

### 4. Request/Response Structure
**File**: `internal/llm/openai/client.go`
- Structure messages as array of role/content objects
- Include self-model as first system message
- Handle `finish_reason` field ("stop", "tool_calls", "length", etc.)
- Validate response structure before parsing
- Use `max_tokens` parameter from config (not unlimited)
- Set `frequency_penalty` and `presence_penalty` if configured

### 5. Token Usage Logging
**File**: `internal/llm/openai/client.go`
- Extract `usage` field from response
- Log prompt_tokens, completion_tokens, total_tokens
- Format: "OpenAI tokens: prompt=X, completion=Y, total=Z"
- Include model name in log for traceability

### 6. Model Selection & Versioning
**File**: `internal/llm/openai/client.go`
- Default to `gpt-4o` if model not specified
- Support model overrides from config
- Validate model name format
- Add comment warning about model deprecation

**File**: `README.md`
- Document recommended models (gpt-4o, gpt-4-turbo)
- Add model version pinning guidance
- Document deprecation handling strategy

## Testing Checklist
- [ ] Streaming responses work correctly
- [ ] Stream handles errors gracefully
- [ ] Tool calls are properly formatted in OpenAI schema
- [ ] Tool responses are parsed correctly
- [ ] Multiple tool calls in one turn work
- [ ] Error handling for each HTTP status code
- [ ] Exponential backoff works for retries
- [ ] Token usage is logged correctly
- [ ] Missing API key shows helpful error
- [ ] Rate limit errors are handled gracefully
- [ ] Model selection works with defaults and overrides

## Success Criteria
- Streaming responses match Ollama streaming quality
- Native OpenAI tool calling replaces hardcoded instructions
- All HTTP error codes handled with clear user feedback
- Token usage visible in logs
- No regression in Phase 1 functionality
- Model selection is flexible and well-documented

## Known Limitations (Phase 2)
- No cost calculation or warnings (Phase 3)
- No configurable retry limits (Phase 3)
- No circuit breaker pattern (Phase 3)
- Basic timeout handling only

## Git Commit Message
```
[openai-integration] Add Phase 2: Production-ready features

- Implement SSE streaming for real-time responses
- Add native OpenAI tool calling (function calling API)
- Comprehensive error handling for all HTTP status codes
- Exponential backoff for rate limits and server errors
- Token usage logging for visibility and debugging
- Model selection with sensible defaults (gpt-4o)
- Update documentation with production usage examples

Phase 2 brings OpenAI backend to production quality with feature
parity to Ollama backend and proper cloud API error handling.
```

## Next Phase
Phase 3 will add optimization features: cost monitoring, advanced retry logic, and timeout management.
