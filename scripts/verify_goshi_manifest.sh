#!/bin/bash
# verify_goshi_manifest.sh
# Verifies source code integrity against .goshi/goshi.manifest

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
MANIFEST="$REPO_ROOT/.goshi/goshi.manifest"

if [ ! -f "$MANIFEST" ]; then
    echo "ERROR: .goshi/goshi.manifest not found"
    echo "Generate it with: scripts/generate_goshi_manifest.sh"
    exit 1
fi

cd "$REPO_ROOT"

hash_file() {
    local target="$1"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$target" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$target" | awk '{print $1}'
    else
        echo "ERROR: No SHA256 tool found (sha256sum or shasum required)"
        exit 1
    fi
}

TOTAL=0
VERIFIED=0
FAILED=0
MISSING=0

while IFS= read -r line; do
    line=$(echo "$line" | sed 's/^ *//;s/ *$//')
    if [ -z "$line" ] || [[ "$line" == \#* ]]; then
        continue
    fi

    if [[ "$line" == TARBALL* ]]; then
        hash=$(echo "$line" | awk '{print $2}')
        path=$(echo "$line" | awk '{print $4}')

        if [ ! -f "$path" ]; then
            echo "MISSING tarball: $path"
            MISSING=$((MISSING + 1))
            continue
        fi

        current_hash=$(hash_file "$path")
        if [ "$current_hash" = "$hash" ]; then
            VERIFIED=$((VERIFIED + 1))
        else
            echo "MISMATCH tarball: $path"
            echo "  Expected: $hash"
            echo "  Got:      $current_hash"
            FAILED=$((FAILED + 1))
        fi
        TOTAL=$((TOTAL + 1))
        continue
    fi

    if [[ "$line" == FILE* ]]; then
        prefix=$(echo "$line" | awk '{print $1}')
        hash=$(echo "$line" | awk '{print $2}')
        size=$(echo "$line" | awk '{print $3}')
        mode=$(echo "$line" | awk '{print $4}')
        mtime=$(echo "$line" | awk '{print $5}')
        file=$(echo "$line" | awk '{for (i=6; i<=NF; i++) printf $i (i==NF?"":" ") }')

        if [ -z "$file" ]; then
            continue
        fi

        if [ ! -f "$file" ]; then
            echo "MISSING file: $file"
            MISSING=$((MISSING + 1))
            TOTAL=$((TOTAL + 1))
            continue
        fi

        current_hash=$(hash_file "$file")
        if [ "$current_hash" = "$hash" ]; then
            VERIFIED=$((VERIFIED + 1))
        else
            echo "MISMATCH file: $file"
            echo "  Expected: $hash"
            echo "  Got:      $current_hash"
            FAILED=$((FAILED + 1))
        fi
        TOTAL=$((TOTAL + 1))
        continue
    fi

    if [[ "$line" == VERSION* ]]; then
        continue
    fi

    echo "WARN: Unrecognized line: $line"
done < "$MANIFEST"

echo "Verified: $VERIFIED"
echo "Missing:  $MISSING"
echo "Failed:   $FAILED"

echo ""
if [ $MISSING -eq 0 ] && [ $FAILED -eq 0 ]; then
    echo "Integrity verification PASSED"
    exit 0
fi

echo "Integrity verification FAILED"
exit 1
