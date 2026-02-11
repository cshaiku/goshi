# OpenAI Integration - Phase 3: Optimization

**Status**: Not Started  
**Phase**: 3 of 3  
**Commit Descriptor**: `[openai-integration]`  
**Date**: 2026-02-10

## Objective
Add optimization features for cost control, reliability, and performance of OpenAI backend.

## Dependencies
- Phase 1 complete (basic OpenAI client)
- Phase 2 complete (streaming, tools, error handling)

## Implementation Tasks

### 1. Cost Monitoring & Warnings
**File**: `internal/llm/openai/cost.go`
- Token-to-cost calculator with model pricing table
- Configurable cost warning thresholds
- Cumulative session cost tracking
- Cost estimation before requests (optional)
- Support for `max_completion_tokens` parameter

**File**: `internal/config/config.go`
- Add OpenAI-specific config section:
  ```yaml
  llm:
    openai:
      max_cost_per_session: 1.00  # USD
      warn_on_high_cost: true
      track_usage: true
  ```

**File**: `internal/llm/openai/client.go`
- Check cost threshold before making requests
- Warn user when approaching limits
- Optional: fail-fast if cost exceeded

### 2. Advanced Retry Logic
**File**: `internal/llm/openai/retry.go`
- Configurable retry limits (default: 3)
- Exponential backoff with jitter
- Backoff strategy: min 1s, max 60s
- Retry decision based on error type:
  - 429 Rate Limit: Always retry
  - 500 Server Error: Retry
  - 503 Service Unavailable: Retry
  - 401 Unauthorized: Never retry
  - Network timeout: Retry with increased timeout
- Log retry attempts with reasoning

**File**: `internal/config/config.go`
- Add retry configuration:
  ```yaml
  llm:
    openai:
      max_retries: 3
      retry_backoff_min: 1  # seconds
      retry_backoff_max: 60 # seconds
  ```

### 3. Circuit Breaker Pattern
**File**: `internal/llm/openai/circuit_breaker.go`
- Track consecutive failures
- Open circuit after N failures (default: 5)
- Half-open state: test with single request
- Closed state: normal operation
- Automatic recovery after cooldown period
- Fallback to Ollama if available (respects local-first)

**File**: `internal/llm/select.go`
- Add circuit breaker state to provider selection
- Auto-failover to Ollama when OpenAI circuit is open
- Log circuit state changes

### 4. Timeout Management
**File**: `internal/llm/openai/client.go`
- Per-request timeout from `request_timeout` config
- Streaming timeout (longer for streaming responses)
- Connection timeout (separate from request timeout)
- Context cancellation support
- Timeout error messages with diagnostics

**File**: `internal/config/config.go`
- Add timeout configuration:
  ```yaml
  llm:
    openai:
      request_timeout: 60      # seconds
      streaming_timeout: 120   # seconds (longer for streams)
      connection_timeout: 10   # seconds
  ```

### 5. Connection Reuse & HTTP Keep-Alive
**File**: `internal/llm/openai/client.go`
- Use `http.Client` with connection pooling
- Configure MaxIdleConns and IdleConnTimeout
- Proper connection cleanup on client close
- Thread-safe client initialization (sync.Once)

### 6. Request Cancellation Support
**File**: `internal/llm/openai/client.go`
- Proper `context.Context` propagation
- Support for user-initiated cancellation
- Clean up resources on cancellation
- Log cancellation events

### 7. Enhanced Configuration & Documentation
**File**: `goshi.yaml`
- Add comprehensive OpenAI configuration example:
  ```yaml
  llm:
    provider: openai
    model: gpt-4o
    
    openai:
      api_key_env: OPENAI_API_KEY
      organization_id_env: OPENAI_ORG_ID  # optional
      
      # Cost control
      max_cost_per_session: 1.00
      warn_on_high_cost: true
      
      # Reliability
      max_retries: 3
      circuit_breaker_threshold: 5
      
      # Timeouts
      request_timeout: 60
      streaming_timeout: 120
  ```

**File**: `README.md`
- Add "Cost Management" section
- Document retry and circuit breaker behavior
- Add troubleshooting guide for OpenAI errors
- Cost estimation examples

**File**: `docs/LLM_INTEGRATION.md`
- Add "OpenAI Backend Architecture" section
- Document cost tracking API
- Explain circuit breaker behavior
- Add advanced configuration examples

## Testing Checklist
- [ ] Cost calculation matches OpenAI pricing
- [ ] Warning shown when approaching cost threshold
- [ ] Retry logic works with exponential backoff
- [ ] Jitter prevents thundering herd
- [ ] Circuit breaker opens after failures
- [ ] Circuit breaker recovers after cooldown
- [ ] Fallback to Ollama works when circuit open
- [ ] Timeouts are respected for all request types
- [ ] Connection pooling reduces latency
- [ ] Context cancellation cleans up resources
- [ ] All configuration options are validated
- [ ] Documentation is clear and comprehensive

## Success Criteria
- Cost visibility prevents surprise bills
- Retry logic handles transient failures gracefully
- Circuit breaker prevents cascading failures
- Timeout management prevents hung requests
- Connection reuse improves performance
- Configuration is flexible and well-documented
- OpenAI backend is fully production-optimized
- No regression in Phase 1 or Phase 2 functionality

## Performance Targets
- Request latency: < 2s for simple queries (excluding LLM processing)
- Streaming latency: < 500ms to first token
- Connection reuse: > 80% of requests use pooled connections
- Circuit recovery: < 2 minutes after issue resolution
- Cost tracking overhead: < 10ms per request

## Git Commit Message
```
[openai-integration] Add Phase 3: Optimization features

- Implement cost monitoring with configurable thresholds
- Add advanced retry logic with exponential backoff and jitter
- Implement circuit breaker pattern with auto-recovery
- Enhanced timeout management for requests and streaming
- Connection pooling and HTTP keep-alive for performance
- Request cancellation support with proper cleanup
- Comprehensive OpenAI configuration options
- Update documentation with cost management and troubleshooting

Phase 3 completes OpenAI integration with production-grade
optimization for cost control, reliability, and performance.
```

## Project Completion
Phase 3 marks the completion of full OpenAI integration. The backend now has:
- ✅ Feature parity with Ollama
- ✅ Production-grade error handling
- ✅ Cost control and monitoring
- ✅ Reliability patterns (retry, circuit breaker)
- ✅ Performance optimization
- ✅ Comprehensive documentation

Next steps: Monitor production usage and iterate based on user feedback.
