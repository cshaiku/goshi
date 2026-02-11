# OpenAI Integration - Phase 1: MVP

**Status**: In Progress  
**Phase**: 1 of 3  
**Commit Descriptor**: `[openai-integration]`  
**Date**: 2026-02-10

## Objective
Implement basic OpenAI backend to enable cloud LLM support alongside Ollama, respecting Goshi's local-first philosophy.

## Self-Model Compliance
- ✅ Maintains local-first execution model (Ollama remains default)
- ✅ OpenAI is fallback/alternative, not primary
- ✅ Respects `authentication: optional: true` from self-model
- ✅ No breaking changes to existing Ollama functionality

## Implementation Tasks

### 1. Create OpenAI Backend Client
**File**: `internal/llm/openai/client.go`
- Implement `Backend` interface (Stream method)
- API key loading from `OPENAI_API_KEY` environment variable
- Basic request structure with messages array
- System prompt integration (inject tool instructions like Ollama)
- Temperature: 0 (deterministic tool calling per Goshi design)
- Basic error handling (connection errors, HTTP status codes)

### 2. Update Backend Selection Logic
**File**: `internal/cli/chat.go`
- Add OpenAI backend instantiation when provider="openai"
- Pass through model configuration
- Graceful error handling with clear user feedback

### 3. Update Provider Selection
**File**: `internal/llm/select.go`
- Ensure OpenAI detection doesn't break auto-detect logic
- Keep Ollama as priority in auto-detect mode

### 4. Configuration Validation
**File**: `internal/config/config.go` (if needed)
- Validate OpenAI-specific settings
- Document OPENAI_API_KEY requirement

### 5. Documentation Updates
**File**: `README.md`
- Add OpenAI setup instructions
- Document OPENAI_API_KEY environment variable
- Add usage examples

**File**: `docs/LLM_INTEGRATION.md`
- Add OpenAI backend examples
- Document differences between Ollama and OpenAI backends

## Testing Checklist
- [ ] OpenAI client connects successfully with valid API key
- [ ] Tool instructions are injected correctly
- [ ] System prompt is sent as first message
- [ ] Error message shown for missing/invalid API key
- [ ] Ollama backend still works (no regression)
- [ ] Provider selection auto-detect prioritizes Ollama
- [ ] Manual OpenAI selection works via config

## Success Criteria
- User can set `provider: openai` in config
- User can set `OPENAI_API_KEY` environment variable
- Basic chat interaction works with OpenAI backend
- Tool instructions are properly formatted for OpenAI
- No breaking changes to existing Ollama functionality

## Known Limitations (Phase 1)
- No streaming support (Phase 2)
- No native OpenAI tool calling (Phase 2)
- No retry logic (Phase 3)
- No cost monitoring (Phase 3)
- Basic error handling only

## Git Commit Message
```
[openai-integration] Add Phase 1: MVP OpenAI backend support

- Create internal/llm/openai/client.go with Backend interface implementation
- Add OPENAI_API_KEY environment variable support
- Update backend selection in chat.go to instantiate OpenAI client
- Maintain local-first philosophy: Ollama remains default
- Add basic error handling for API connection issues
- Update documentation with OpenAI setup instructions

Phase 1 provides minimal viable OpenAI integration while preserving
all existing Ollama functionality and Goshi's safety invariants.
```

## Next Phase
Phase 2 will add production-ready features: streaming, native tool calling, and comprehensive error handling.
