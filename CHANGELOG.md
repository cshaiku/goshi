# Changelog

All notable changes to goshi are documented in this file. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

## [1.0.0] - 2026-02-10

### Added - LLM Integration Refactoring (Feb 10, 2026)

#### Phase 1-3: Core LLM Architecture (Feb 10)
- **Structured Message Types** — Type-safe message protocol with `UserMessage`, `AssistantTextMessage`, `AssistantActionMessage`, `ToolResultMessage`, and `SystemPrompt` types
- **Tool Registry** — Dynamic tool discovery with JSON Schema validation, auto-populated from action dispatcher
- **Permission Model** — Fine-grained capability-based access control with `FS_READ` and `FS_WRITE` capabilities
- **Audit Trail** — Complete logging of all permission decisions with timestamps and context
- **Chat Session Manager** — Encapsulated conversation state management with message history
- **6-Phase Chat Flow** — Deterministic execution pipeline (Listen → Detect → Plan → Parse → Act → Report)
- **Ollama Backend** — Tool instruction generation and structured output parsing

#### Phase 4: Integration Testing (Feb 10)
- **18 Comprehensive Integration Tests** covering all critical chat and tool execution paths
- **MockLLMBackend** — Deterministic testing infrastructure for scripted responses
- **End-to-End Validation** — Full 6-phase flow testing with permission enforcement, schema validation, and audit trail verification
- Test coverage across tool execution, permission denial scenarios, schema validation, structured response parsing, and message history

#### Phase 5: Documentation (Feb 10)
- **README.md Enhancements** — New LLM Integration Architecture section with tool registry, permission model, and chat session documentation
- **MIGRATION.md** — Comprehensive upgrade guide with step-by-step migration instructions, breaking changes, and troubleshooting
- **LLM_INTEGRATION.md** — Quick reference guide with code examples, tool documentation, and usage patterns
- **PHASES_COMPLETE.md** — Detailed completion summary of all 5 LLM integration phases with test breakdown and achievements

### Added - CLI Enhancements (Feb 10, 2026)

#### Command Structure & Help (Feb 10)
- **fs probe → fs probe subcommand** — Standardized command structure for consistency
- **Comprehensive Exit Codes** — Full EXIT CODES documentation across all CLI commands
- **Standardized Environment Variables** — Consistent ENVIRONMENT documentation across all commands (with doctor command additions)
- **Long Description** — Comprehensive long description for root goshi command 
- **Help Text Improvements** — Streamlined fs and config command help for clarity

#### Output Format Standardization (Feb 10)
- **--format Flag** — Unified `--format (json|yaml|human)` flag for doctor and heal commands
- **JSON Output Deprecation** — Deprecated `--json` flag in favor of unified `--format` approach

### Added - Core Testing Infrastructure (Feb 10, 2026)

- **Config Package Tests** (51 tests) — Configuration validation, environment handling, parameter bounds
- **Filesystem Safety Tests** (13 tests) — Path traversal protection, symlink handling, guard mechanisms
- **Protocol Tests** (8 tests) — Request parsing, manifest validation, JSON handling
- **Detection Tests** (7 tests) — Binary detection, PATH handling
- **Diagnosis Tests** (6 tests) — Issue creation, severity assignment
- **Execution Tests** (9 tests) — Dry-run vs actual execution, error handling
- **Verification Tests** (8 tests) — Pass/fail determination, failure reporting
- **LLM Integration Tests** (78+ tests across all phases)

**Total: 165+ tests (100% passing)**

### Added - Project Management (Feb 10, 2026)

- **Action Plans** — CLI UX improvement and project management action plans in `.goshi/action-plans`
- **Documentation Strategy** — Integrated threat model into self-model and human context
- **Test Coverage Commitments** — Updated self-model and human context with comprehensive testing commitments

### Changed - Documentation (Feb 10, 2026)

- **README.md** — Updated execution pipeline documentation and test coverage information
- **Threat Model Integration** — Incorporated threat model into self-model and human context strategy
- **Self-Model & Human Context** — Enhanced with test coverage commitments and safety validation strategy

### Metadata

- **Build Status** — ✅ All passing
- **Test Coverage** — 165+ tests across 15+ packages
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
- See individual phase documentation files (MIGRATION.md, LLM_INTEGRATION.md, PHASES_COMPLETE.md) for technical details
