# Graph runtime qualification — 2026-10-15

Bridge 1.2.0 / graph protocol 1.1.0. Shared package authored in public Bram and
mirrored with SHA-256 receipts. No private Vigil source is included in public
snapshots. The public releases retain each tool's native authority boundaries.

Local macOS checks: all six Go suites and vet; shared-package race detection;
installed cross-tool planning, imports, selective resume, evidence verification,
large integer preservation and tamper rejection. Nora's shared package also
passes with its declared Go 1.22 toolchain. These checks use owned fixtures.

Native Windows 11 x64 and Debian Linux x64 run the same standalone graph tests:
checkpoint interruption/resume, single-writer locking and process-death release,
active cancellation, bounded output, stale inputs, SDK identity changes, output
tampering and golden schema contracts. Test executables are digest-checked before
execution in unique temporary directories; no installed tools are replaced.

- Windows: passed; test executable SHA-256 `45524f22d8db79f4aff822ded9e4fcd5287ec91d5493a19c5fbe683c38c8655c`.
- Linux: passed; test executable SHA-256 `31acd9ab949888b73815cfe8539d408a60d402e1d9671fbbd2d7c8cc35cc39ec`.

CI now runs shared graph contracts and native recovery tests on macOS, Linux and
Windows in each repository, plus native CLI capability/schema discovery. This
configuration does not by itself establish hosted CI success. The all-six
installed compatibility script is a separate local integration check.

Qualification applies to the declared graph runtime paths. It does not establish
all tools' native installers, live AI providers, SDK closure, application UI,
physical-venue behavior, or remote service reproducibility. AI diagnosis remains
explicit and advisory. Shell hooks are opt-in and limited to the tools namespace.

Release follow-up: the standard-library vulnerability gate on Go 1.26.6 requires
Go 1.26.9. Release builds select that patched toolchain through go.mod. Existing
language minimums remain unchanged except Clyde’s explicit installer pin.
