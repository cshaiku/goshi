# Security Testing for Goshi

## Overview

Goshi implements an **offensive security testing** framework to validate the integrity checking system. These tests intentionally tamper with source files to ensure that the integrity diagnostics correctly detect unauthorized modifications.

## Why Offensive Testing?

Traditional unit tests verify that code works under normal conditions. Offensive security tests go further by:

1. **Simulating real attacks**: Intentionally modifying files to test detection
2. **Validating defense mechanisms**: Ensuring integrity checks actually work
3. **Building confidence**: Proving the system detects tampering, not just assuming it does
4. **Red team approach**: Thinking like an attacker to find weaknesses

## Architecture

### Build Tag Isolation

Offensive tests are isolated from normal CI runs using Go build tags:

```go
//go:build offensive
```

This ensures that:
- Regular `go test ./...` skips offensive tests (fast feedback)
- Offensive tests only run when explicitly requested
- CI pipelines can choose when to run security validation

### Safe Tampering Pattern

All offensive tests follow a **backup/restore pattern** for guaranteed safety:

```go
// Tamper with a file
restore, err := helper.TamperWithFile(filePath)
if err != nil {
    t.Fatal(err)
}
defer restore()  // Always restored, even on test failure

// Run verification
result := diagnostic.Run()

// Assert tampering was detected
if !detectedTampering(result) {
    t.Error("Failed to detect file tampering!")
}
```

Key safety features:
- **Backup before modification**: Original content saved to temp file
- **Deferred restoration**: `defer` ensures cleanup even on panic/failure
- **No git modifications**: Tests work on working tree, not committed files
- **Temporary files**: All test artifacts cleaned up automatically

## Test Scenarios

### 1. File Tampering Detection (`TestIntegrityDetectsTampering`)

**What it does**: Randomly selects a `.go` file from the manifest, modifies it by adding a comment, then verifies the integrity checker detects the hash mismatch.

**Why it matters**: Ensures that even subtle changes (like adding comments) are detected.

```bash
go test -tags=offensive -v ./internal/diagnostics/integrity/ -run TestIntegrityDetectsTampering
```

### 2. Missing File Detection (`TestIntegrityDetectsMissingFile`)

**What it does**: Temporarily deletes a random file from the manifest, verifies detection, then restores it.

**Why it matters**: Validates detection of deleted or moved files.

```bash
go test -tags=offensive -v ./internal/diagnostics/integrity/ -run TestIntegrityDetectsMissingFile
```

### 3. Clean Repository Validation (`TestIntegrityPassesWhenClean`)

**What it does**: Runs integrity check on a clean repository without modifications.

**Why it matters**: Ensures no false positives - clean repos should pass.

```bash
go test -tags=offensive -v ./internal/diagnostics/integrity/ -run TestIntegrityPassesWhenClean
```

### 4. Multiple Modifications (`TestMultipleModifications`)

**What it does**: Tampers with 2 files simultaneously, verifies both are detected.

**Why it matters**: Tests that the system catches multiple concurrent attacks.

```bash
go test -tags=offensive -v ./internal/diagnostics/integrity/ -run TestMultipleModifications
```

### 5. Severity Validation (`TestSeverityLevels`)

**What it does**: Verifies that integrity issues are reported with ERROR severity.

**Why it matters**: Ensures proper severity classification for security issues.

```bash
go test -tags=offensive -v ./internal/diagnostics/integrity/ -run TestSeverityLevels
```

## Running Offensive Tests

### Locally

Run all offensive tests:
```bash
go test -tags=offensive -v ./internal/diagnostics/integrity/
```

Run specific test:
```bash
go test -tags=offensive -v ./internal/diagnostics/integrity/ -run TestIntegrityDetectsTampering
```

Run with race detection:
```bash
go test -tags=offensive -race -v ./internal/diagnostics/integrity/
```

### CI/CD

Offensive tests run in GitHub Actions on the `offensive-tests` job:

```yaml
offensive-tests:
  name: Offensive Security Tests
  runs-on: ubuntu-latest
  steps:
    - name: Run offensive tests
      run: go test -tags=offensive -v ./internal/diagnostics/integrity/
```

This job runs:
- ✅ On every push to main
- ✅ On pull requests
- ✅ On scheduled cron (weekly validation)

## Test Utilities

### `TestHelper`

Located in [`internal/diagnostics/integrity/testutil.go`](../internal/diagnostics/integrity/testutil.go), provides:

- **`RandomGoFile()`**: Selects random `.go` file from manifest
- **`TamperWithFile(path)`**: Modifies file, returns restore function
- **`DeleteFile(path)`**: Temporarily deletes file, returns restore function
- **`BackupFile(path)`**: Creates backup in temp directory
- **`RestoreFile(path, backup)`**: Restores from backup and cleans up

Example usage:
```go
helper, err := NewTestHelper()
if err != nil {
    t.Fatal(err)
}

// Select a random target
targetFile, err := helper.RandomGoFile()
if err != nil {
    t.Fatal(err)
}

// Tamper with it (automatically backed up)
restore, err := helper.TamperWithFile(targetFile)
if err != nil {
    t.Fatal(err)
}
defer restore()  // Guaranteed restoration

// Now verify detection works
// ...
```

## Integration with Doctor Command

The integrity diagnostics are integrated into the `goshi doctor` command:

```bash
$ goshi doctor
Detected issues:
 - [error][INTEGRITY_HASH_MISMATCH] 1 files have been modified:
   internal/cli/doctor.go
     Expected: 2d1793f963d57b22...
     Actual:   f7647020e14bbff3...
   (suggested: Review changes and regenerate goshi.sum after committing valid changes.)
```

JSON output:
```bash
$ goshi doctor --format=json
{
  "Issues": [
    {
      "Code": "INTEGRITY_HASH_MISMATCH",
      "Message": "1 files have been modified:\n  internal/cli/doctor.go\n    Expected: 2d1793f963d57b22...\n    Actual:   f7647020e14bbff3...",
      "Strategy": "Review changes and regenerate goshi.sum after committing valid changes.",
      "Severity": "error"
    }
  ]
}
```

## Best Practices

### For Test Writers

1. **Always use `defer restore()`**: Never rely on manual cleanup
2. **Test on random files**: Use `RandomGoFile()` for realistic scenarios
3. **Verify restoration**: Add assertions that files are actually restored
4. **Document intent**: Explain what attack you're simulating

### For CI Maintainers

1. **Run offensive tests regularly**: Weekly cron job recommended
2. **Monitor for flakiness**: File system operations can be platform-specific
3. **Check test duration**: Offensive tests should complete in < 5 seconds
4. **Alert on failures**: Security test failures are critical

### For Developers

1. **Run before major changes**: Validate integrity system still works
2. **Add tests for new attacks**: If you think of a new attack vector, test it
3. **Keep manifest updated**: Regenerate `goshi.sum` after file changes
4. **Review security issues**: Don't ignore integrity check failures

## Troubleshooting

### "File not found" errors

**Cause**: Test selected a file that was recently deleted or moved.
**Solution**: Regenerate `goshi.sum` to update the manifest.

```bash
bash scripts/generate_goshi_sum.sh
```

### Tests hang or timeout

**Cause**: File restoration may be failing, leaving locks.
**Solution**: Check for orphaned backup files in `/tmp/`:

```bash
ls -la /tmp/goshi_backup_*
rm /tmp/goshi_backup_*  # Clean up if needed
```

### "Repository not clean" failures

**Cause**: Previous test didn't restore properly.
**Solution**: Reset working tree:

```bash
git checkout HEAD -- .
git clean -fd
```

## Security Considerations

### What Offensive Tests Don't Do

- ❌ Modify git history
- ❌ Commit changes
- ❌ Push to remote
- ❌ Modify files outside the repository
- ❌ Require elevated privileges

### What They Do Protect

- ✅ Validate integrity checking works
- ✅ Detect tampering before deployment
- ✅ Build confidence in security posture
- ✅ Document expected behavior under attack

## Future Enhancements

Potential additions to the offensive testing framework:

1. **Timing attacks**: Verify detection happens within acceptable time
2. **Concurrent modifications**: Test race conditions in integrity checking
3. **Partial file corruption**: Modify middle of file instead of appending
4. **Permission changes**: Test detection of file mode changes
5. **Symlink attacks**: Test handling of symbolic links
6. **Binary tampering**: Extend to test binary integrity in bin/

## Related Documentation

- [Contributing Guide](../CONTRIBUTING.md)
- [Threat Model](../goshi.threat.model.md)
- [Self Model](../goshi.self.model.yaml)
- [LLM Integration](../LLM_INTEGRATION.md)

## Contact

For security concerns or questions about offensive testing:
- Open an issue: https://github.com/cshaiku/goshi/issues
- Review threat model: [goshi.threat.model.md](../goshi.threat.model.md)
