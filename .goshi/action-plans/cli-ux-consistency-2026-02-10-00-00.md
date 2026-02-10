# Goshi CLI UX Consistency Action Plan

**Created:** 2026-02-10-00-00

## Goals
- Improve user experience and consistency across all goshi CLI commands.

## Tasks

1. **Command Naming Consistency**
   - Evaluate moving `fs-probe` under `fs probe` or grouping experimental commands under an `experimental` or `dev` parent.
   - If not moved, document rationale in help text.

2. **Output Format Flag Standardization**
   - Standardize output format flags across commands (e.g., always use `--format=json|yaml` or always use `--json`).
   - Update help text and examples accordingly.

3. **Help Text Improvements**
   - Shorten overly long help texts; move advanced details to a `SEE ALSO` or `--long-help` section.
   - Ensure all commands have a concise summary and clear usage examples.

4. **Subcommand Discovery & Grouping**
   - Group experimental or advanced commands under a clear parent (e.g., `experimental` or `dev`).
   - Ensure all subcommands are discoverable via `--help`.

5. **Exit Code Documentation**
   - Add exit code documentation to all commands that may fail, for scripting users.

6. **Environment Variable Documentation**
   - Standardize the placement and order of environment variable documentation in all help outputs.

## Implementation Steps
- [ ] Review and refactor command structure for naming and grouping.
- [ ] Refactor output format flags for consistency.
- [ ] Edit help texts for brevity and clarity.
- [ ] Add or update exit code documentation.
- [ ] Standardize environment variable documentation.
- [ ] Test all commands for UX and help output.
- [ ] Commit changes with a summary referencing this action plan.

---
