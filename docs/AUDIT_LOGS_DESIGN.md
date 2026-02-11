# Audit Logs UX and Architecture Design

## Purpose

Provide a clear, user-friendly audit trail for all security-relevant actions in goshi. The audit trail must be:
- Easy to view and export
- Safe (redaction by default)
- Deterministic and inspectable
- Consistent across CLI and TUI
- Non-invasive to existing workflows

## Goals

- Surface permission grants/denials clearly and consistently.
- Capture tool execution attempts and outcomes.
- Provide quick, human-readable summaries for users.
- Allow structured export for automation (JSON/YAML).
- Keep audit data local-first with explicit retention controls.

## Non-Goals

- Remote telemetry or cloud upload.
- Real-time streaming outside the local machine.
- Automatic log sharing without explicit user action.

## Feature Set (Fleshed Out)

### 1) Audit Event Types

- Permission decision
  - GRANT, DENY, AUTO_CONFIRM
- Tool invocation
  - Tool name, args summary, validation status, execution status
- Safety/guard events
  - Guard deny, unsafe path, scope violation
- Diagnostics events
  - Integrity mismatches, missing files, unsafe state
- Session lifecycle
  - Session start/end, model provider selected

### 2) Human-Readable Timeline

- Chronological timeline view
- Compact, single-line entries for scanning
- Expandable details for each entry
- Optional grouping by type (permissions, tools, safety)

Example (human view):
```
[14:32:11] GRANT  FS_READ   (user-approved) in /Users/cs/goshi
[14:32:20] TOOL   fs.list   ok  path="."  (5 items)
[14:33:04] DENY   FS_WRITE  (user-denied) in /Users/cs/goshi
[14:34:19] INTEGRITY ERROR  goshi.sum mismatch
```

### 3) Structured Export

- JSON and YAML export
- JSONL for streaming append (per event, one line)
- Schema versioning for future compatibility

Example JSONL record:
```
{"ts":"2026-02-10T14:32:20Z","type":"tool","name":"fs.list","status":"ok","cwd":"/Users/cs/goshi","args":{"path":"."}}
```

### 4) Search and Filters

- Filter by type: permissions, tool, safety, diagnostics
- Filter by status: ok, warn, error
- Filter by tool name or capability
- Time window: --since, --until
- Limit count: --limit

### 5) Redaction and Privacy

- Redact secrets in args and env values
- Redaction enabled by default
- Tool argument visibility is configurable via `tool_arguments_style`
- Logs are stored locally and remain readable (no encryption by default)

### 6) Local Persistence and Retention

- Default location: .goshi/audit/
- Per-session files: session-<id>.jsonl
- Session-specific logs only (no global aggregate)
- Retention policy: keep last N sessions or N days
- Explicit user command to purge

## UX Surface

### CLI: `goshi audit`

Command options:
- `--format=human|json|yaml`
- `--latest` (default)
- `--session=<id>`
- `--since=1h|2026-02-10T12:00:00Z`
- `--limit=200`
- `--type=permission,tool,safety,diagnostic`
- `--status=ok,warn,error`
- `--unsafe` (no redaction)

Example:
```
# Last 50 events
 goshi audit --limit=50

# Last hour, only tool events
 goshi audit --since=1h --type=tool

# Export JSON
 goshi audit --format=json --session=latest
```

### TUI

- Add an Audit panel view
- Default to last 20 events
- Key bindings:
  - / for filter
  - Enter to expand details
  - Tab to cycle Audit, Inspect, Output

## Data Model (Proposed)

```go
type AuditEvent struct {
  Timestamp time.Time `json:"ts"`
  Type      string    `json:"type"` // permission, tool, safety, diagnostic, session
  Action    string    `json:"action"` // GRANT, DENY, TOOL_CALL, TOOL_OK, TOOL_ERR
  Status    string    `json:"status"` // ok, warn, error
  Message   string    `json:"message"`
  Cwd       string    `json:"cwd"`
  Details   map[string]any `json:"details,omitempty"`
  SessionID string    `json:"session_id"`
  Version   string    `json:"version"` // schema version
}
```

## Integration Points

- Permissions audit already exists; extend to emit `AuditEvent`.
- ToolRouter: log validated tool calls and execution results.
- Diagnostics: log integrity, safety, and guard results.
- Session start/end: create session events on init/cleanup.

## Config (Proposed)

```
audit:
  enabled: true
  dir: .goshi/audit
  retention_days: 14
  max_sessions: 50
  redact: true
  tool_arguments_style: summaries
```

## Risks and Mitigations

- Risk: Logs contain secrets
  - Mitigation: redaction by default; allow explicit override
- Risk: Performance cost
  - Mitigation: buffered writes; JSONL append
- Risk: Log bloat
  - Mitigation: retention policies; limits

## Milestones

1) Schema + writer (JSONL, redaction)
2) CLI read/export
3) Permissions/tool/diagnostic integration
4) TUI panel
5) Config + retention

## Decisions

- Tool argument visibility defaults to summaries. Config options: full | long | short | summaries.
- Logs are session-specific only (one file per session).
- Logs remain local and readable; encryption is not required by default.
