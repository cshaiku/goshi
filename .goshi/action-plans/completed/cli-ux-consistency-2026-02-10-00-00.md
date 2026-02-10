<!--
  FILE STATUS DOCBLOCK
  ====================
  Status: COMPLETED
  Completion Date: 2026-02-10
  Completed By: Automated CLI UX Improvement Sprint
  Duration: Single session (2026-02-10)
  
  Related Commits:
  - 6d456f0: Command structure: Move fs-probe to fs probe subcommand for consistency
  - 85c9fe6: docs: Add EXIT CODES documentation to all CLI commands
  - ec61c75: docs: Standardize ENVIRONMENT variable documentation across commands
  
  Completion Notes:
  - Moved fs-probe from top-level command to 'goshi fs probe' subcommand
  - Streamlined help text for fs and config commands for clarity
  - Added EXIT CODES sections to all CLI commands documenting exit codes for scripting users
  - Added ENVIRONMENT sections standardizing variable documentation across all commands
  - All changes tested and verified with bin/goshi --help
  
  Summary:
  Comprehensive CLI UX improvements including command structure refactoring,
  help text reorganization, and documentation standardization. All goals achieved
  across five related task categories.
-->

# Goshi CLI UX Consistency Action Plan

**Created:** 2026-02-10-00-00  
**Status:** ✅ COMPLETED

## Goals
- Improve user experience and consistency across all goshi CLI commands.

## Tasks

1. **Command Naming Consistency** ✅
   - Moved `fs-probe` under `fs probe` as a subcommand.
   - Improves discoverability and command grouping.

2. **Output Format Flag Standardization** ✅
   - Standardized output format flags across commands using `--format=json|yaml|human`.
   - Completed in separate action plan (output-format-flags).

3. **Help Text Improvements** ✅
   - Streamlined fs and config command help texts for brevity and clarity.
   - Maintained examples and usage information.

4. **Subcommand Discovery & Grouping** ✅
   - fs-probe now properly grouped under fs command.
   - All subcommands discoverable via `--help`.

5. **Exit Code Documentation** ✅
   - Added EXIT CODES sections to all commands that may fail.
   - Completed in separate action plan (exit-code-docs).

6. **Environment Variable Documentation** ✅
   - Standardized placement and order of environment variable documentation.
   - Completed in separate action plan (env-var-docs).

## Implementation Summary
- ✅ Reviewed and refactored command structure for naming and grouping.
- ✅ Refactored output format flags for consistency.
- ✅ Edited help texts for brevity and clarity.
- ✅ Added exit code documentation.
- ✅ Standardized environment variable documentation.
- ✅ Tested all commands for UX and help output.
- ✅ Committed changes with clear references to action plans.

---
