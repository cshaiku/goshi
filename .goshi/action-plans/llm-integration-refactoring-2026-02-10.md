# LLM Integration Refactoring - Master Plan

**Status:** Not Started  
**Created:** 2026-02-10  
**Author:** Development Team  
**Target:** Modernize chat loop and LLM integration with structured outputs and OpenAI best practices

---

## Overview

Current goshi LLM integration suffers from architectural inconsistencies that undermine safety and auditability:
- Dual tool detection paths (regex + LLM parsing)
- Ambiguous message types and unclear conversation semantics
- Fragile tool call parsing (substring-based JSON extraction)
- Conflicting streaming UX and structured response needs
- Inconsistent permission enforcement across two checkpoints
- Free-form text mixing reasoning with action declarations

**Goal:** Implement a structured agentic chat loop following OpenAI best practices, providing deterministic tool use, clear audit trails, and extensible architecture.

---

## Guiding Principles (OpenAI Standards)

1. **Function Calling over Raw Text** — Tools explicitly declared with schemas; LLM returns structured calls not freeform JSON
2. **Single Source of Truth** — One tool registry; permissions enforced once at declaration time
3. **Structured Messages** — Type-safe message format with explicit roles and content types
4. **Streaming + Structured** — Stream for UX responsiveness but collect and validate complete structures before action
5. **Fail-Closed** — Ambiguity, parsing errors, or permission issues halt execution immediately
6. **Traceable Decisions** — Every tool call, permission grant, and denial is logged and auditable
7. **Test-Driven** — All tool flows covered by integration tests before merge

---

## Work Breakdown

### Phase 1: Architecture & Types (Foundation)
- [ ] **1.1** Create `StructuredMessage` type system (`internal/llm/messages.go`)
  - Replaces generic `Message` with typed variants: `UserMessage`, `AssistantMessage`, `AssistantAction`, `ToolResult`, `ErrorMessage`
  - Includes message validation and serialization

- [ ] **1.2** Build `ToolRegistry` system (`internal/app/tool_registry.go`)
  - Centralized tool definition with JSON schemas, descriptions, required permissions
  - Tool lookup, validation, and permission pre-checks
  - Replaces ad-hoc tool detection

- [ ] **1.3** Define OpenAI-style structured output (`internal/llm/structured_output.go`)
  - Response can be: text message, action request, error, or tool result
  - Clear type discrimination for parsing and routing

### Phase 2: Backend Updates (LLM Integration)
- [ ] **2.1** Update Ollama backend for structured outputs (`internal/llm/ollama/client.go`)
  - Modify tool instruction prompt to request JSON Schema compliant responses
  - Handle structured output parsing from LLM
  - Maintain streaming capability

- [ ] **2.2** Implement robust structured response parsing (`internal/llm/structured_parser.go`)
  - Full JSON Schema validation before accepting tool calls
  - Clear error messages for parse failures
  - Defer to LLM for retry if invalid

### Phase 3: Chat Loop Refactoring (Core Logic)
- [ ] **3.1** Redesign permission model (`internal/cli/permissions.go`)
  - Single grant-time check, not per-tool verification
  - Audit logging for all permission decisions
  - Permission inheritance and scoping

- [ ] **3.2** Refactor main chat loop (`internal/cli/chat.go`)
  - Remove regex pre-detection, use LLM for all tool decisions
  - Implement explicit phases: listen → plan → act → report
  - Structured UI output (streaming planning, then action outcome)
  - Remove duplicate tool calling code

- [ ] **3.3** Create chat session manager (`internal/cli/session.go`)
  - Encapsulate message history, permissions, context
  - Provide clean interface for chat operations
  - Enable replay and testing

### Phase 4: Testing & Validation
- [ ] **4.1** Integration tests for tool flows (`internal/cli/chat_test.go`)
  - Test each tool call path end-to-end
  - Mock LLM for deterministic testing
  - Permission enforcement validation

- [ ] **4.2** Structured output validation tests (`internal/llm/structured_parser_test.go`)
  - Schema validation edge cases
  - Malformed response recovery
  - Error message clarity

### Phase 5: Documentation & Migration
- [ ] **5.1** Update README with new paradigm (`README.md`)
  - Document structured output format
  - Tool registry discovery
  - Permission model

- [ ] **5.2** Create migration guide for integrations
  - How tools are now declared
  - New message format
  - Breaking changes reference

---

## Success Criteria

1. ✅ Single unified tool definition system (no regex duplication)
2. ✅ 100% of tool calls validated against JSON schema before execution
3. ✅ All permission decisions logged and auditable
4. ✅ Chat loop phases clearly separated and testable
5. ✅ Streaming UX preserved (character-by-character display during planning)
6. ✅ All edge cases covered by tests (89+ new tests)
7. ✅ Zero tool call parsing failures in test suite
8. ✅ Clear audit trail for each interaction

---

## Risk Mitigations

| Risk | Mitigation |
|------|-----------|
| Breaking existing tool calls | Implement compatibility layer; deprecation period before removal |
| Streaming compatibility | Collect complete structured response before parsing, stream planning text |
| Performance regression | Profile before/after; optimize schema validation path |
| Test coverage gaps | Mandate tests for each tool before merge to main |

---

## Related Documents

- `goshi.self.model.yaml` — Safety invariants and fail-closed semantics
- `docs/goshi.threat.model.md` — T5 (Behavior Drift) directly addressed by this refactoring
- Individual action plans in `.goshi/action-plans/llm-*`

---

## Notes

- This refactoring directly addresses Threat T5 (Behavior Drift) by making all reasoning and decisions explicit and traceable
- Aligns with OpenAI function calling paradigm which has proven robust in production
- Enables future enhancements: tool use logging, user feedback on decisions, multi-turn tool chains
