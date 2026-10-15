# Shared graph workflows

Local implementation, 2026-10-15. Bridge 1.2.0 / graph protocol 1.1.0.

Vigil, Loom, Clyde, Bram, Nora and Goshi expose the same `tools graph` contract.
Each keeps its native engine and can be built/used without the other tools.
Companions are needed only when a workflow names them. This feature is local;
installing it does not publish a release or start an AI provider.

## Explain, plan, run, resume, verify

```sh
vigil tools graph capabilities
loom tools graph schema --kind workflow
vigil tools graph plan --repo . --workflow workflow.json > /tmp/review-plan.json
loom tools graph explain --plan /tmp/review-plan.json
clyde tools graph context --plan /tmp/review-plan.json --stage investigate
```

Inspect the exact argv, input paths, missing tools and warnings. Copy the plan's
`digest` into the explicit execution request:

```sh
vigil tools graph run --plan /tmp/review-plan.json --run-dir /tmp/NEW-REVIEW --approve-digest REVIEWED_DIGEST
loom tools graph explain --run-dir /tmp/NEW-REVIEW
clyde tools graph verify --run-dir /tmp/NEW-REVIEW
bram tools graph resume --plan /tmp/review-plan.json --run-dir /tmp/NEW-REVIEW --approve-digest REVIEWED_DIGEST
```

All invocations above refer to the same schema and checkpoint state. A run
requires a new directory outside the inspected repository, with an existing
parent. Do not reuse example output-directory names accidentally. Run directories
are private on POSIX (0700, files 0600); no artifacts are automatically deleted.
Planning, explanation, context selection, schema discovery and verification
execute no companion commands and write only stdout/stderr.

After inputs change, create and review a new plan, then resume the existing run
with the new digest. Intact successful nodes with matching fingerprints are reused;
changed nodes and their dependency descendants are invalidated. Unrelated node
inputs do not invalidate independent work. Workflow-wide inputs, context and
imports conservatively invalidate every node. A change to one node's argv only
invalidates it and its descendants.

## Workflow declaration

```json
{
  "schema": "starlit.workflow.v1",
  "name": "ui-review",
  "budget_seconds": 300,
  "inputs": ["ui/Main.qml"],
  "environment": ["QT_PLUGIN_PATH"],
  "context": [
    {"path": "docs/ui-owner.md", "stages": ["investigate", "implement"]},
    {"path": "docs/verification.md", "stages": ["verify"]}
  ],
  "nodes": [
    {
      "id": "components",
      "kind": "analysis",
      "argv": ["loom", "graph:components", "ui", "--format", "json"],
      "inputs": ["ui/Shared.qml"],
      "scopes": [{"root": "ui", "suffixes": [".qml"]}],
      "timeout_seconds": 30,
      "max_attempts": 1,
      "replay_safe": true
    },
    {
      "id": "policy",
      "kind": "check",
      "argv": ["python3", "tests/check_ui.py"],
      "inputs": ["tests/check_ui.py"],
      "needs": ["components"],
      "timeout_seconds": 60,
      "max_attempts": 2,
      "replay_safe": true
    }
  ]
}
```

Paths in `inputs`/`context` are clean repository-relative regular files.
`scopes` expand selected directories by suffix and track membership, so additions
and deletions also invalidate affected nodes. Each scope accepts `root`, `suffixes`
and optional `exclude_dirs` names; `.git` is always excluded. Discovery is bounded
at 32,768 entries per scope. Scopes can be workflow-wide or node-specific. Declare
scripts/configuration and every source dependency required for a result's reuse;
the engine does not infer a complete build closure. SHA-256 fingerprints include
those bytes, dependency fingerprints, exact node definitions, selected executable
path/bytes, OS/architecture, repository location, PATH and explicitly named
environment values (including PATH and PATHEXT). Environment values are hashed, not printed. Tool interpreters,
dynamic libraries, SDKs, remote services and undeclared environment/state are not
automatically included. A shell interpreter argv is still an explicit executable
request; the runner itself never interpolates command strings.

Node kinds are `check`, `analysis`, and `ai`. Successful analysis is advisory;
`declared_checks_passed` requires successful declared check nodes. Even that label
means only the declared checks passed for the tracked input scope. It does not
certify all product behavior or every platform.

## Connect native reports

```sh
loom graph:components ui --format json > /tmp/components.json
clyde bundle . --include 'ui/*.qml' --out /tmp/NEW-BUNDLE
clyde bundle verify /tmp/NEW-BUNDLE
vigil impact:plan --json > /tmp/impact.json
vigil tools graph plan --repo . --workflow workflow.json \
  --loom /tmp/components.json \
  --clyde-manifest /tmp/NEW-BUNDLE/manifest.json \
  --impact /tmp/impact.json > /tmp/connected-plan.json
```

Select imports relevant to your repository; none is mandatory.

- **Loom:** imports component/use relationships, unresolved identities and source
  fingerprints. Current Loom emits `source_files` hashes and detects a file
  changing during extraction. Duplicate component names become explicit ambiguous
  diagnostics. Legacy graphs without producer hashes carry a freshness warning.
  Relationships remain heuristic, not a complete language-level call graph.
- **Clyde:** imports manifest source identities and checks current bytes against
  recorded hashes. This does not replace native bundle/chunk verification, secret
  scanning or upload approval. No source is uploaded by the graph engine.
- **Vigil:** imports owner/context relationships and check prerequisites. Checks
  with explicit `argv` become executable plan nodes; legacy command-only advice
  stays a warning. Imported checks default to one non-replayable attempt. Review
  their potentially mutating behavior before approving execution. Unmatched paths
  remain a coverage warning. The impact producer remains advisory.

The knowledge entities/links and executable workflow are distinct fields of one
plan. Imports and their declared source identities are fingerprinted. Stale
producer hashes fail planning instead of silently qualifying new source.

## Recovery, retries and verification

Commands run sequentially in dependency order. A failed prerequisite blocks its
dependents while independent work may continue. Per-node timeouts, a workflow
execution budget (default 300 seconds, maximum 24 hours), bounded output and a
maximum of three attempts prevent open-ended loops. Retrying requires explicit
`replay_safe: true`; a repeated identical failure stops the current retry loop.
Completed attempt durations count against the budget on resume. An interrupted
attempt also consumes an attempt slot; its unrecorded elapsed time is not known.

State is atomically replaced after checkpoints; attempts/plans/raw outputs are
immutable. Kernel file locks prevent concurrent writers and release on process
exit. Interrupted replay-safe checks can resume within their remaining budget.
Attempted non-replayable operations require manual recovery before repetition;
the engine does not guess whether an upload/publish completed or replay it.
Orphaned files from an interrupted write are retained. This is not a power-loss
or hostile concurrent-filesystem durability guarantee.

`verify` checks stored plan identity, node fingerprints, artifact/output digests,
and current declared inputs/tools. It returns stale or failed status with a
nonzero exit. Plan approval and consistency hashes are not signatures or proof
of authorship. Outputs preserve exact bytes, including JSON integers. Inspect
raw outputs when evaluating what a check actually established.

## Bounded AI diagnosis

AI is optional. Declare a selected report or a failure-only diagnostic node:

```json
{
  "id": "diagnose-policy",
  "kind": "ai",
  "peer": "YOUR_CONFIGURED_BRAM_PEER",
  "prompt": "Explain the failure and suggest a minimal repair for review.",
  "report_from": "policy",
  "when": "failure",
  "needs": ["policy"],
  "timeout_seconds": 120,
  "max_attempts": 1,
  "replay_safe": false
}
```

Run also requires `--allow-ai`. Failure reports contain at most 256 KiB of stdout,
64 KiB stderr and attempt metadata. Successful checks skip failure-only diagnosis.
Alternatively `report` selects an explicitly declared repository-relative file
of at most 1 MiB, matching Bram’s native input bound.
The selected Bram peer may send this content to an external provider and incur
costs. The engine never chooses a peer or starts a model daemon implicitly.

A diagnosis records the exact failed-attempt identity. If that evidence is
superseded on resume, the old diagnosis is blocked rather than silently reused
or automatically sent to a peer again.

AI outputs cannot be workflow prerequisites, never count as checks, and cannot
turn a failed check into success. Suggested repairs are not automatically applied.
Bounded retries repeat approved replay-safe commands; they are not autonomous
source-editing loops. Configured peer commands have their own privileges: asking
for diagnosis is not a sandbox or restriction on a peer's implementation.

Goshi's execution surface accepts only its own doctor/version diagnostics and
JSON flags. It gains no general repository repair or AI authority through this
shared bridge. Other tools' native mutation, upload and credential requirements
remain in force.

## Interfaces and limits

`capabilities` publishes contract names and limits. `schema --kind` emits structural
JSON Schema for workflow/plan/run/verification; the CLI also validates semantic
bounds, dependencies and cycles. JSON output and structured errors have stable
schema identifiers. `explain` additionally supports text and Mermaid. Exit codes:
0 success, 1 failed/stale execution or verification, 2 invalid request/plan, 130
user interrupt. CLI `--help` and the existing shell command structure still work.

Vigil also advertises read-only `graph:capabilities`, `graph:plan`, `graph:explain`,
`graph:context`, `graph:verify` through its registry/MCP. They suppress automatic
evidence/timing writes. Execution remains the explicit native `tools graph`
request; there is no new MCP execution/approval bypass.

Limits: 128 nodes, 16 imports, 4096 input paths, 8192 knowledge entities, 32768
links, 8 MiB per JSON document/file, 256 MiB per executable, 64 MiB total inputs, 2 MiB stdout and 64 KiB
stderr per attempt. Context excerpts are UTF-8 prefixes, at most 4 KiB per file
and 32 KiB total; stages choose which documents are included. Read full owner
sections where excerpts truncate them. Special files, input symlinks and path
escapes are refused. Local directory aliases such as macOS `/tmp` are canonicalized.
Commands inherit the caller's environment and privileges; timeout/cancellation
bounds capture, not every descendant process or external effect.

## Implementation and qualification

The shared 0BSD implementation/tests live in Bram's `internal/toolbridge`.
Other CLIs carry digest-checked snapshots; use Bram's `scripts/sync-toolbridge.py`
for explicit updates. No private Vigil implementation is copied to public tools.

Regression tests exercise selective reuse, stale/tampered evidence, dependency
blocking, recovery, budgets, output limits, context stages and explicit mock AI
feedback. Real multi-tool compatibility and local qualification are recorded with
this work's validation results. No external AI/upload or native Windows/Linux
runtime qualification is implied by local macOS checks.

## Local qualification — 2026-10-15

All six native Go suites pass on this Mac, and the shared engine passes the Go
race detector. `scripts/test-graph-compatibility.py` in Bram checks six installed
capability contracts, identical connected-plan digests, native Loom/Clyde imports,
cross-tool run/resume/verification, selective stale-input recovery, Goshi authority
and output-tamper detection. AI tests use a local mock; no real peer/upload runs.
The shared package cross-compiles for Windows/Linux amd64; native runtime behavior
on those systems is not newly qualified.

The owned Voci source-review workflow executes Loom analysis, Clyde reporting and
seven QML policy tests. Its first analysis failure exposed QML lexer limitations;
the retained run resumes after the parser fix and reuses the independent Clyde
result. Expanded Voci build/UI recommendations are planned, not executed by this
source-review qualification. Heuristic/unresolved graph warnings remain explicit.
No whole-app, physical-device, token-savings or autonomous-repair claim is made.

## Configuration and runtime freshness (protocol 1.1.0)

Declare tool-specific dependencies by the exact executable name used in node argv:

```json
"tool_inputs": [{
  "name": "python3",
  "files": ["pyproject.toml"],
  "optional_files": ["~/.config/example/config.json"],
  "runtimes": ["python3"],
  "environment": ["PYTHONPATH"]
}]
```

Required files must exist. Optional files track absence, appearance and deletion.
Files may be repository-relative, absolute, or begin with `~/`. Declare a small
SDK identity file instead of an SDK tree. Runtime entries identify executable
bytes; they do not enumerate dynamic libraries. Changed tool inputs invalidate
that tool's consumers and their dependants. Other nodes remain reusable.
External paths appear in plans; only environment values are hashed. This is an
explicit dependency contract, not automatic discovery or hermetic execution.

## Shell completion and compatibility

`TOOL tools completion bash|zsh|fish|powershell` prints an opt-in shell hook for
the `tools` namespace. Source the output using the shell's normal mechanism.
These hooks may replace an existing completer for the command; review the output
before installing it. `TOOL tools complete -- graph context --stage ''` is a pure
completion query; it does not execute tools, read configuration, or contact AI.

Each repository runs the same golden JSON schema contracts and graph tests on
macOS, Linux and Windows CI. Bram's `scripts/test-graph-compatibility.py` also
checks all six installed CLIs together locally. Native runtime qualification
uses only an owned standalone test executable in remote temporary directories;
`scripts/qualify-graph-native.py` records digests, platform and test results.
