# Changelog

All notable changes to goshi are documented in this file. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Planned
- Enhanced LLM backend selection and fallback logic
- Extended tool registry with custom tool registration
- Multi-session conversation management

---

## [1.1.0] - 2026-02-10

### Added - Text User Interface (TUI)

#### Complete Interactive Terminal Interface
- **Bubble Tea Framework** — Modern TUI built with Bubble Tea v1.3.10, Lipgloss v1.1.0, Bubbles v1.0.0
- **Real-time LLM Streaming** — Async streaming with progressive message display and streaming cursor (▊)
- **Tool Execution** — Full tool execution with visual feedback (✓ success, ✗ failure indicators)
- **Scrollable Chat History** — Viewport component with keyboard navigation (↑/↓ arrows)
- **User Input** — Textarea component with Ctrl+S to send, Ctrl+C/Esc to quit
- **Message Styling** — Color-coded messages (orange user, green assistant) with Lipgloss
- **Status Display** — Header with law metrics, constraints, and enforcement status
- **Performance Optimization** — 100 message viewport limit to prevent memory issues
- **Enhanced Error Handling** — Truncated error messages, nil-safe operations, user-friendly feedback
- **Comprehensive Testing** — 14 unit tests covering initialization, streaming, tool execution, error handling
- **Mode Detection** — `--headless` flag for CLI/script mode (TUI is default)
- **Session Management** — New `internal/session` package for shared state (broke cli/tui import cycles)
- **Complete Documentation** — 312-line guide in `.goshi/ai-related/TUI_USAGE.md` with architecture, keyboard shortcuts, troubleshooting

#### Default Mode Change (Breaking)
- **TUI Mode** — Running `goshi` without arguments now launches interactive TUI (previously CLI)
- **CLI/Script Mode** — Use `goshi --headless` for traditional command-line mode
- **Updated Help Text** — Documentation now clearly distinguishes between TUI and headless modes

### Changed - Documentation
- **README.md** — Added Interactive Modes section documenting TUI and headless modes with keyboard controls
- **Test Count** — Updated from 165+ to 230+ tests (verified accurate count)

### Fixed - Test Infrastructure
- **metrics_test.go** — Recreated corrupted test file with proper structure
- **session package** — Exported MockStream types for test reuse across packages
- **cli integration tests** — Updated imports to use session.NewChatSession

### Metadata
- **Build Status** — ✅ All passing
- **Test Coverage** — 230+ tests across 16+ packages
- **TUI Framework** — Bubble Tea (Elm Architecture pattern)

---

## [1.0.0] - 2026-02-10

### Added - OpenAI Backend Integration

#### Phase 1: MVP OpenAI Backend Support
- **OpenAI API Client** — Full OpenAI chat completions API integration with streaming support
- **Environment Configuration** — API key management via `OPENAI_API_KEY` environment variable
- **Model Selection** — Support for all OpenAI models (default: gpt-4o-mini)
- **Tool Calling Integration** — Structured JSON tool instructions compatible with OpenAI format
- **Error Handling** — HTTP error handling with descriptive user messages

#### Phase 2: Production-Ready Features
- **SSE Streaming** — Server-Sent Events streaming for real-time response delivery
- **Retry Logic** — Exponential backoff with configurable retry attempts (default: 3 retries)
- **Retryable Error Detection** — Smart classification of transient vs permanent API errors (429, 500, 502, 503, 504)
- **Request/Response Logging** — Comprehensive error logging with retry attempt tracking

#### Phase 3: Optimization Features
- **Connection Pooling** — HTTP client with MaxIdleConns (100) and keep-alive support for connection reuse
- **Cost Tracking** — Per-model token usage tracking with configurable warning ($1) and max ($10) thresholds
- **Circuit Breaker** — Fault tolerance with automatic failure detection (5 failures) and 30s cooldown
- **Performance Logging** — Real-time cost and token usage reporting per request

#### OpenAI Test Suite
- **68 Comprehensive Tests** covering all Phase 3 components with 100% pass rate
- **Cost Tracker Tests** (14 tests) — Usage recording, warning thresholds, cost limits, concurrent access, per-model pricing
- **Circuit Breaker Tests** (21 tests) — State transitions (Closed → Open → Half-Open), recovery logic, concurrent access
- **Error Handling Tests** (10 tests) — Retryable error codes, exponential backoff calculation, HTTP error handling
- **SSE Streaming Tests** (14 tests) — Chunk parsing, usage data extraction, finish reasons, malformed JSON handling
- **Tool Conversion Tests** (20 tests) — OpenAI format conversion, schema validation, complex nested structures

### Added - LLM Integration Refactoring

#### Phase 1-3: Core LLM Architecture
- **Structured Message Types** — Type-safe message protocol with `UserMessage`, `AssistantTextMessage`, `AssistantActionMessage`, `ToolResultMessage`, and `SystemPrompt` types
- **Tool Registry** — Dynamic tool discovery with JSON Schema validation, auto-populated from action dispatcher
- **Permission Model** — Fine-grained capability-based access control with `FS_READ` and `FS_WRITE` capabilities
- **Audit Trail** — Complete logging of all permission decisions with timestamps and context
- **Chat Session Manager** — Encapsulated conversation state management with message history
- **6-Phase Chat Flow** — Deterministic execution pipeline (Listen → Detect → Plan → Parse → Act → Report)
- **Ollama Backend** — Tool instruction generation and structured output parsing

#### Phase 4: Integration Testing
- **18 Comprehensive Integration Tests** covering all critical chat and tool execution paths
- **MockLLMBackend** — Deterministic testing infrastructure for scripted responses
- **End-to-End Validation** — Full 6-phase flow testing with permission enforcement, schema validation, and audit trail verification
- Test coverage across tool execution, permission denial scenarios, schema validation, structured response parsing, and message history

**Total: 165+ tests (100% passing)**

### Changed - Code Quality Improvements

#### SOLID Principles Refactoring
- **Dependency Inversion Principle (DIP)** — Extracted `BackendFactory` to decouple CLI from concrete LLM implementations (ollama, openai)
- **Single Responsibility Principle (SRP)** — Extracted `PermissionHandler` to isolate permission request logic from main chat loop
- **Open/Closed Principle (OCP)** — New LLM backends can be added to factory without modifying existing chat logic
- **Magic Strings Elimination** — Moved ANSI color codes to `DisplayConfig` with centralized constants and optional color support
- **God Object Reduction** — Simplified `runChat()` by 65 lines through delegation to specialized handlers

#### New Abstractions
- **`internal/cli/backend_factory.go`** — Factory pattern for LLM backend instantiation (54 lines)
- **`internal/cli/display.go`** — Display configuration and color management (29 lines)
- **`internal/cli/permission_handler.go`** — Permission request logic encapsulation (66 lines)

#### Bug Fixes
- **Duplicate Package Declarations** — Fixed syntax errors in `internal/repair/basic_test.go` and `internal/selfmodel/metrics_test.go`

### Changed - Documentation
- **README.md** — Updated with LLM Integration Architecture section with tool registry, permission model, and chat session documentation
- **MIGRATION.md** — Comprehensive upgrade guide with step-by-step migration instructions, breaking changes, and troubleshooting
- **LLM_INTEGRATION.md** — Quick reference guide with code examples, tool documentation, and usage patterns
- **Internal Phase Documentation** — Detailed completion summary of all 5 LLM integration phases (internal tracking, see docs/ai-patterns/project-phases/)

### Added - CLI Enhancements

#### Command Structure & Help
- **fs probe → fs probe subcommand** — Standardized command structure for consistency
- **Comprehensive Exit Codes** — Full EXIT CODES documentation across all CLI commands
- **Standardized Environment Variables** — Consistent ENVIRONMENT documentation across all commands (with doctor command additions)
- **Long Description** — Comprehensive long description for root goshi command 
- **Help Text Improvements** — Streamlined fs and config command help for clarity

#### Output Format Standardization
- **--format Flag** — Unified `--format (json|yaml|human)` flag for doctor and heal commands
- **JSON Output Deprecation** — Deprecated `--json` flag in favor of unified `--format` approach

### Added - Core Testing Infrastructure

- **Config Package Tests** (51 tests) — Configuration validation, environment handling, parameter bounds
- **Filesystem Safety Tests** (13 tests) — Path traversal protection, symlink handling, guard mechanisms
- **Protocol Tests** (8 tests) — Request parsing, manifest validation, JSON handling
- **Detection Tests** (7 tests) — Binary detection, PATH handling
- **Diagnosis Tests** (6 tests) — Issue creation, severity assignment
- **Execution Tests** (9 tests) — Dry-run vs actual execution, error handling
- **Verification Tests** (8 tests) — Pass/fail determination, failure reporting
- **LLM Integration Tests** (78+ tests across all phases)

**Total: 165+ tests (100% passing)**

### Added - Project Management

- **Action Plans** — CLI UX improvement and project management action plans in `.goshi/action-plans`
- **Documentation Strategy** — Integrated threat model into self-model and human context
- **Test Coverage Commitments** — Updated self-model and human context with comprehensive testing commitments

### Changed - Documentation

- **README.md** — Updated execution pipeline documentation and test coverage information
- **Threat Model Integration** — Incorporated threat model into self-model and human context strategy
- **Self-Model & Human Context** — Enhanced with test coverage commitments and safety validation strategy

### Metadata

- **Build Status** — ✅ All passing
- **Test Coverage** — 165+ tests across 15+ packages
- **LLM Backends** — Ollama (local), OpenAI (cloud)
- **Deployment Ready** — Yes
- **Breaking Changes** — Documented in MIGRATION.md

---

## Previous Work

### Project Origins

goshi is designed as a constrained, local-first model of AI-assisted tooling with maximum safety and auditability. The project explores bounded autonomy with:

- Explicit, machine-enforced understanding of scope and identity
- Pre-action safety validation
- Auditable filesystem proposals
- Deterministic, inspectable diagnostics
- Self-healing constraints limited to the tool's own repository

### Core Features (Pre-1.0)

- diagnostics-first execution pipeline with safety phases
- Two-phase, proposal-based filesystem model
- Path traversal protection and symlink handling
- Comprehensive environment detection
- Issue diagnosis with severity classification
- Action repair planning and execution
- Verification and dry-run support

---

## Future Releases

### Planned Features

- Enhanced LLM backend selection and fallback logic
- Extended tool registry with custom tool registration
- Multi-session conversation management
- Persistent audit trail storage
- Tool execution history and rollback capabilities
- Custom permission models and RBAC
- Performance profiling and optimization

---

## Notes

- All dates reflect development date (2026-02-10)
- Semantic versioning: [MAJOR.MINOR.PATCH]
- See git log for detailed commit history
- See migration guides: [MIGRATION.md](MIGRATION.md) for upgrade instructions, [LLM_INTEGRATION.md](LLM_INTEGRATION.md) for quick reference
