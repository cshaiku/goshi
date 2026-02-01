# goshi

goshi is a Go-based CLI tool and the self-hosted, self-aware successor to grok-cli.

Its primary goal is to be safe, diagnosable, and self-healing — for itself only.

---

## Purpose

goshi explores a stricter model of AI-assisted tooling where:

- The tool has an explicit, machine-enforced understanding of what it is
- Safety invariants are checked before any action
- Self-healing is constrained strictly to the tool’s own repository
- Diagnostics are deterministic, automatable, and human-readable

This is an experiment in bounded autonomy, not a general-purpose agent.

---

## Core Concepts

### Human Context
Declares intent and purpose.

File:
- goshi.human.context.yaml

### Self Model
Defines machine-enforced identity, scope, and safety constraints.

File:
- goshi.self.model.yaml

### Diagnostics-First Execution
All actions are gated by diagnostics phases, executed in order:

1. Safety invariants
2. Self-model compliance
3. Environment checks

If any phase fails, execution halts.

---

## Commands

### Diagnostics

goshi diagnostics  
goshi diagnostics --json

Characteristics:
- Verdict-first output
- Deterministic exit codes
- JSON suitable for automation

Exit codes:
- 0 — success
- 2 — self-model violation
- 3 — safety invariant violation

---

## Safety Model (High Level)

goshi enforces the following invariants:

- Binary identity must be goshi
- Execution must occur within its own repository
- No filesystem mutation outside allow-listed paths
- No healing when the git working tree is dirty
- No execution with unexpected privileges

Violations are treated as safety breaches, not errors.

---

## Non-Goals

- Managing or healing other projects
- Acting as a general AI agent
- Modifying user systems outside the repository
- Silent or implicit behavior changes

---

## Status

Early / Experimental

Safety and self-model layers are under active development.  
Interfaces may evolve; the safety philosophy will not.

---

## License

MIT
