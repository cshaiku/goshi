# WORK IN PROGRESS #

# goshi

**goshi** is a Go-based CLI tool and the self-hosted, self-aware successor to grok-cli.

Its primary goal is to be **safe, diagnosable, auditable, and self-healing — for itself only**.

This project explores a constrained, local-first model of AI-assisted tooling where **no action is implicit and no mutation is silent**.

---

## Purpose

goshi explores a stricter model of AI-assisted automation where:

- The tool has an explicit, machine-enforced understanding of what it is
- Safety invariants are checked before *any* action
- Filesystem mutation is gated behind explicit, auditable proposals
- Self-healing is constrained strictly to the tool’s own repository
- Diagnostics and decisions are deterministic and inspectable

This is an experiment in **bounded autonomy**, not a general-purpose AI agent.

---

## Core Concepts

### Human Context

Declares intent and purpose.

File:
- `goshi.human.context.yaml`

---

### Self Model

Defines machine-enforced identity, scope, and safety constraints.

File:
- `goshi.self.model.yaml`

The self model is treated as **authoritative** and violations are considered safety breaches.

---

### Diagnostics-First Execution

All actions are gated by diagnostics phases, executed in order:

1. Safety invariants
2. Self-model compliance
3. Environment checks

If any phase fails, execution halts.

---

## Filesystem Safety Model

goshi uses a **two-phase, proposal-based filesystem model**.

### Key Properties

- **No filesystem mutation happens immediately**
- All writes are first recorded as proposals
- Proposals are persisted and auditable
- Applying a proposal requires explicit confirmation
- Dry-run is enabled by default

---

### Write Proposals

Creating a write **does not modify the filesystem**.

```bash
echo "NEW CONTENT" | goshi fs write path/to/file.txt
```

---

### Applying a Proposal

Applying a proposal **requires two explicit opt-ins**:

```bash
goshi fs apply <proposal-id> --yes --dry-run=false
```

---

### Drift Protection

If the target file has changed since the proposal was created, apply will fail.

---

## Execution Pipeline

goshi follows a deterministic, auditable pipeline:

1. **Detect** — Scan environment for binaries, configuration issues, and safety concerns
2. **Diagnose** — Analyze detected issues and assign severity levels  
3. **Repair** — Plan corrective actions 
4. **Execute** — Apply actions (dry-run by default)
5. **Verify** — Confirm repairs were successful

Each phase is independent, deterministic, and inspectable.

---

## Testing

goshi includes comprehensive test coverage for reliability and security:

- **Config** (51 tests): Configuration validation, environment variable handling, parameter bounds
- **Filesystem Safety** (13 tests): Path traversal protection, symlink handling, guard mechanisms
- **Protocol** (8 tests): Request parsing, manifest validation, JSON handling
- **Detection** (7 tests): Binary detection, PATH handling
- **Diagnosis** (6 tests): Issue creation, severity assignment
- **Execution** (9 tests): Dry-run vs actual execution, error handling
- **Verification** (8 tests): Pass/fail determination, failure reporting

**Total: 89+ passing tests** across core packages.

Run all tests:

```bash
go test ./internal/...
```

Run tests for a specific package:

```bash
go test -v ./internal/config
```

---

## Build & Run

```bash
go build -o goshi
./goshi --help
```

Build with make:

```bash
make build
```

---

## License

MIT
