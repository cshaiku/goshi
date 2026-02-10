# Goshi CLI Output Format Flag Consistency Action Plan

**Created:** 2026-02-10-00-00

## Goal
Standardize output format flags (e.g., --format=json|yaml|human) across all goshi CLI commands for a consistent user experience.

## Tasks
- [x] Refactor doctor and heal commands to use --format (json|yaml|human) and deprecate --json.
- [x] Restore --dry-run and --yes flags for heal.
- [ ] Update help text and examples for all affected commands.
- [ ] Test all affected commands for correct flag behavior and output.
- [ ] Commit changes with a summary referencing this action plan.

---
