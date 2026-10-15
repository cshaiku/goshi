# Optional CLI integration

Normal commands keep their own engines, configuration and installation. Companion tools are needed only for recipes that name them. The shared 0BSD bridge is compiled into each CLI; there is no central daemon or shared package to install. JSON is the common report envelope; each tool keeps its existing native JSON/GCF/text interfaces.

## Inventory and planning

```sh
goshi tools doctor
goshi tools identity
goshi tools plan repo-review --repo /path/to/repo
goshi tools plan repo-review --repo /path/to/repo --ui /path/to/view.qml --with-vigil
```

Doctor checks PATH, binary digests, both receipt formats, source commits and tracked dirty state. It queries the optional identity command with bounded capture; older companion versions can remain usable without reporting that identity. Discovery commands are tool-specific hints, not proof that an old installation supports every feature. Missing companions do not affect normal commands. Rebuild deliberately from the selected checkout; doctor never installs, repairs or switches Vigil variants.

Plan prints exact argument arrays, input file hashes and availability. It does not execute checks or contact an AI peer.

## Local report workflows

```sh
goshi tools run repo-review --repo /path/to/repo --output-dir /path/to/new-review
goshi tools run ui-review --ui /path/to/view.qml --output-dir /path/to/new-ui-report
goshi tools run log-review --log /path/to/access.log --output-dir /path/to/new-log-report
```

Repository review uses clyde scan-report; optional UI input adds loom inspect:source. Explicit --with-vigil adds the selected Vigil's repo:health --dry-run so its cache is not written. UI review uses loom. Log review uses local nora analysis with IP anonymization. None of these recipes starts AI, SSH, upload or installation operations. These recipes collect evidence; they are not source-bundle creation or a release qualification suite.

Run requires a new output directory with an existing parent, refuses existing directories/links, and stores private plan, per-step artifacts, exact stdout sidecars and summary files (0700 directory, 0600 files on POSIX). Choose a directory outside the inspected repository. Step artifacts preserve exact JSON numeric values, original stderr, exit status, duration and executable/output digests. Failed or missing tools produce failed artifacts and a nonzero workflow exit; other steps still run. Local checks have a 30-second limit, 2 MiB stdout and 64 KiB stderr capture. UI/log/report inputs are regular files bounded at 128 MiB; feedback input has a stricter 1 MiB bound. An input hash change during execution fails the summary. No artifacts are automatically deleted. Process cancellation does not provide a sandbox or complete descendant-process isolation.

## Explicit AI feedback

```sh
goshi tools plan feedback-review --report /path/to/review/01-clyde.json --peer YOUR_PEER
goshi tools run feedback-review --report /path/to/review/01-clyde.json --peer YOUR_PEER --output-dir /path/to/new-feedback
bram ask --peer YOUR_PEER --input /path/to/review/01-clyde.json 'Explain this report'
cat /path/to/report.json | bram ask --peer YOUR_PEER --input - 'Explain this report'
```

Feedback sends the explicitly selected file through the running local bram daemon to that configured peer, which may contact an external provider. It is a separate recipe and is never inferred from local review. The report path stays in argv; its contents travel on stdin to the peer, not in process arguments. Feedback has a two-minute workflow limit, with the peer's own configured timeout still applying. Clyde's NotebookLM upload/digest approvals are not replaced or bypassed. Goshi's repair authority remains limited to itself. The private Vigil variant is never distributed through these public snapshots. Vigil registry/MCP discovery exposes help only; workflow execution is a native CLI feature and does not add an MCP mutation route.

## Snapshot ownership

The standalone bridge is authored in Starlit-Digital/bram internal/toolbridge. Other repos carry the 0BSD source/tests and a SHA-256 UPSTREAM.json receipt. Snapshot tests detect drift. From the bram checkout, scripts/sync-toolbridge.py --repo /path/to/another/repo refreshes an explicitly selected, unchanged snapshot; it refuses local snapshot edits. No Go dependency on another CLI repo is required. Each CLI can be built, installed and used independently.

## Graph workflows

`tools graph` adds shared explain/plan/context/run/resume/verify contracts.
See [graph workflows](GRAPH_WORKFLOWS.md) for input scopes, recovery, evidence and AI boundaries.
