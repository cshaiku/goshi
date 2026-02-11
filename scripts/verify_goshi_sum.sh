#!/bin/bash
# verify_goshi_sum.sh
# Verifies source code integrity against goshi.sum manifest

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
MANIFEST="$REPO_ROOT/goshi.sum"

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║         GOSHI SOURCE INTEGRITY VERIFICATION                    ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Check if manifest exists
if [ ! -f "$MANIFEST" ]; then
    echo "❌ ERROR: goshi.sum not found"
    echo ""
    echo "Generate it with: scripts/generate_goshi_sum.sh"
    exit 1
fi

# Change to repo root
cd "$REPO_ROOT"

echo "[1/3] Loading manifest..."

# Extract metadata
GIT_COMMIT=$(grep "^# Git Commit:" "$MANIFEST" | cut -d: -f2- | xargs)
GIT_BRANCH=$(grep "^# Git Branch:" "$MANIFEST" | cut -d: -f2- | xargs)
GENERATED=$(grep "^# Generated:" "$MANIFEST" | cut -d: -f2- | xargs)

echo "  Manifest commit: $GIT_COMMIT"
echo "  Manifest branch: $GIT_BRANCH"
echo "  Generated: $GENERATED"
echo ""

echo "[2/3] Verifying file hashes..."

# Count total hashes
TOTAL=$(grep "^SHA256 " "$MANIFEST" | wc -l | tr -d ' ')
VERIFIED=0
FAILED=0
MISSING=0

# Verify each hash
while IFS= read -r line; do
    # Parse hash line: SHA256 <hash> <file>
    algo=$(echo "$line" | awk '{print $1}')
    expected_hash=$(echo "$line" | awk '{print $2}')
    file=$(echo "$line" | awk '{$1=""; $2=""; print $0}' | xargs)
    
    # Check if file exists
    if [ ! -f "$file" ]; then
        echo "  ❌ MISSING: $file"
        MISSING=$((MISSING + 1))
        continue
    fi
    
    # Compute current hash
    if command -v sha256sum &> /dev/null; then
        current_hash=$(sha256sum "$file" | awk '{print $1}')
    elif command -v shasum &> /dev/null; then
        current_hash=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        echo "  ERROR: No SHA256 tool found"
        exit 1
    fi
    
    # Compare hashes
    if [ "$current_hash" = "$expected_hash" ]; then
        VERIFIED=$((VERIFIED + 1))
        if (( VERIFIED % 10 == 0 )); then
            echo -ne "  Verified $VERIFIED/$TOTAL files\r"
        fi
    else
        echo "  ❌ MISMATCH: $file"
        echo "     Expected: $expected_hash"
        echo "     Got:      $current_hash"
        FAILED=$((FAILED + 1))
    fi
    
done < <(grep "^SHA256 " "$MANIFEST")

echo "  Verified $VERIFIED/$TOTAL files    "
echo ""

echo "[3/3] Summary..."
echo "  ✓ Verified: $VERIFIED"
if [ $MISSING -gt 0 ]; then
    echo "  ❌ Missing: $MISSING"
fi
if [ $FAILED -gt 0 ]; then
    echo "  ❌ Failed:  $FAILED"
fi
echo ""

# Final result
if [ $MISSING -eq 0 ] && [ $FAILED -eq 0 ]; then
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║              ✅ INTEGRITY VERIFICATION PASSED                  ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    echo "All source files match expected hashes."
    echo "Build is reproducible and untampered."
    exit 0
else
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║              ❌ INTEGRITY VERIFICATION FAILED                  ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    echo "Source code integrity check failed!"
    echo ""
    if [ $MISSING -gt 0 ]; then
        echo "Missing files: $MISSING"
        echo "  → Files in manifest but not found on disk"
    fi
    if [ $FAILED -gt 0 ]; then
        echo "Hash mismatches: $FAILED"
        echo "  → Files have been modified since manifest generation"
    fi
    echo ""
    echo "Possible causes:"
    echo "  1. Source files modified without regenerating manifest"
    echo "  2. Uncommitted changes in working directory"
    echo "  3. Code tampering or corruption"
    echo ""
    echo "To fix: scripts/generate_goshi_sum.sh"
    exit 1
fi
