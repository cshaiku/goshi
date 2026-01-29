# Threat Model: grokgo

## 1. Scope

This threat model applies **only** to grokgo itself.

grokgo is not intended to:
- protect user data
- secure remote systems
- operate as a sandbox
- act as a privileged system agent

The model focuses on **preventing unintended or unsafe autonomous behavior**.

---

## 2. Assets to Protect

| Asset                   | Rationale                                  |
|------------------------|---------------------------------------------|
| Host filesystem        | Prevent unintended modification             |
| User trust             | Tool must remain predictable and bounded    |
| grokgo source integrity| Prevent self-corruption                     |
| Execution intent       | Prevent behavior drift                      |
| Git history            | Preserve recoverability and auditability    |

---

## 3. Trusted Boundaries

| Boundary        | Trust Level                                   |
|-----------------|-----------------------------------------------|
| grokgo binary   | Trusted if identity invariants pass            |
| Repo root       | Trusted operational scope                     |
| Self-model      | Trusted machine-enforced contract             |
| OS / filesystem| Untrusted beyond allow-listed paths           |
| User input     | Always untrusted                              |

---

## 4. Threat Actors

| Actor                    | Capability                                  |
|--------------------------|----------------------------------------------|
| Accidental user error    | Misuse, wrong directory, bad flags            |
| Tool misconfiguration   | Broken env, partial checkout                  |
| Instruction drift       | AI-guided misuse or scope expansion           |
| Malicious local user    | Attempts to coerce external writes            |
| Compromised dependency  | Unexpected behavior via tools                |

---

## 5. Primary Threats and Mitigations

### T1: Scope Escalation
Modification of files outside the grokgo repository.

**Mitigations**
- Path allow-listing
- Absolute + symlink-resolved path checks
- Single filesystem write gate
- Safety invariant hard stop

---

### T2: Identity Confusion
Renamed or copied binaries acting as grokgo.

**Mitigations**
- Binary name invariant
- Repo root marker
- Self-model presence check

---

### T3: Unsafe Self-Healing
Healing runs in an ambiguous or unsafe state.

**Mitigations**
- Safety invariants before healing
- Clean git working tree requirement
- No healing on invariant failure

---

### T4: Privilege Abuse
Execution with elevated or unexpected privileges.

**Mitigations**
- Execution user invariant
- Root disallowed by default
- No implicit privilege escalation

---

### T5: Silent Behavior Drift
Behavior changes without user visibility.

**Mitigations**
- Verdict-first output
- Deterministic exit codes
- JSON and human output parity
- No silent healing

---

## 6. Threat Map (Code-Level)

| Threat | Enforcement Location |
|------|----------------------|
| T1 Scope Escalation | internal/fs/safefs.go |
| T2 Identity Confusion | internal/diagnostics/safety/binary_name.go |
| T3 Unsafe Healing | internal/diagnostics/safety/runner.go |
| T4 Privilege Abuse | internal/diagnostics/safety/user.go |
| T5 Behavior Drift | cmd/grokgo/diagnostics.go |

---

## 7. Explicit Non-Threats

| Excluded Threat     | Reason                                |
|--------------------|----------------------------------------|
| Remote attackers   | No network exposure                    |
| Sandbox escape     | grokgo is not a sandbox                |
| Supply-chain attack| Dependencies assumed trusted (v1)      |
| OS-level exploits | Delegated to the OS                    |
| Malicious models  | Bounded by enforced invariants          |

---

## 8. Failure Philosophy

grokgo uses **fail-closed semantics**.

- Any ambiguity halts execution
- Any invariant violation is unsafe
- No partial success states
- No override flags in early versions

---

## 9. Security Posture Summary

| Property         | Status   |
|------------------|----------|
| Least privilege  | Enforced |
| Scope limitation | Enforced |
| Determinism      | Enforced |
| Recoverability  | High     |
| Autonomy        | Bounded  |

---

## Appendix A: What Would Make grokgo Unsafe

The following changes would violate grokgo’s safety model:

- Allowing filesystem writes outside the repo root
- Introducing multiple filesystem write paths
- Making safety invariants optional or suppressible
- Adding override flags for invariant violations
- Allowing healing with a dirty git working tree
- Executing without verifying binary identity
- Performing network operations without explicit scope
- Silent behavior changes without diagnostics output
- Allowing privilege escalation or root execution by default

Any of the above would collapse the bounded autonomy model.
