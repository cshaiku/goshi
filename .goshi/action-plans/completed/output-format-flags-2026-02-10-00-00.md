<!--
  FILE STATUS DOCBLOCK
  ====================
  Status: COMPLETED
  Completion Date: 2026-02-10
  Completed By: Automated CLI UX Improvement Sprint
  Duration: Single session (2026-02-10)
  
  Related Commits:
  - 3d087d4: Output format flags: Standardize --format (json|yaml|human) for doctor and heal, deprecate --json
  
  Completion Notes:
  - Refactored doctor and heal commands to use --format flag (json|yaml|human)
  - Deprecated --json flag in favor of --format=json
  - Restored --dry-run and --yes flags for heal command
  - Updated help text and examples for affected commands
  - Tested all affected commands for correct flag behavior and output
  - Verified JSON, YAML, and human-readable output formats
  
  Summary:
  Successfully standardized output format flags across doctor and heal commands,
  providing consistent user experience and enabling scripting use cases.
  All tasks completed and changes merged to main.
-->

# Goshi CLI Output Format Flag Consistency Action Plan

**Created:** 2026-02-10-00-00  
**Status:** ✅ COMPLETED

## Goal
Standardize output format flags (e.g., --format=json|yaml|human) across all goshi CLI commands for a consistent user experience.

## Tasks
- ✅ Refactor doctor and heal commands to use --format (json|yaml|human) and deprecate --json.
- ✅ Restore --dry-run and --yes flags for heal.
- ✅ Update help text and examples for all affected commands.
- ✅ Test all affected commands for correct flag behavior and output.
- ✅ Commit changes with a summary referencing this action plan.

## Implementation Summary
- Refactored `goshi doctor` command to use `--format=json|yaml|human` instead of `--json`
- Refactored `goshi heal` command to use `--format=json|yaml|human` instead of `--json`
- Restored `--dry-run` and `--yes` flags for heal command
- Updated config command to support `--format` flag for consistency
- Updated help text and examples for all affected commands
- All changes tested and verified with multiple output formats
- Changes committed and merged to main

---
