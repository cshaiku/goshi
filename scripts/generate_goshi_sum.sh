#!/bin/bash
# generate_goshi_sum.sh
# Generates integrity hashes for all Go source files in the project
# This ensures build reproducibility and detects unauthorized code changes

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OUTPUT_FILE="$REPO_ROOT/goshi.sum"

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║         GOSHI SOURCE INTEGRITY HASH GENERATOR                  ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Change to repo root
cd "$REPO_ROOT"

# Get current git state
if git rev-parse --git-dir > /dev/null 2>&1; then
    GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
    GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
    GIT_TAG=$(git describe --tags --exact-match 2>/dev/null || echo "none")
    GIT_DIRTY=$(git diff --quiet || echo " (dirty)")
else
    GIT_COMMIT="no-git"
    GIT_BRANCH="no-git"
    GIT_TAG="none"
    GIT_DIRTY=""
fi

BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION=$(go version | awk '{print $3}')

# Create header
cat > "$OUTPUT_FILE" << EOF
# goshi.sum - Source Code Integrity Manifest
# Generated: $BUILD_DATE
# Git Commit: $GIT_COMMIT
# Git Branch: $GIT_BRANCH
# Git Tag: $GIT_TAG$GIT_DIRTY
# Go Version: $GO_VERSION
#
# Format: <algorithm> <hash> <file>
# Algorithm: SHA256
#
# This file contains cryptographic hashes of all Go source files
# to ensure build reproducibility and detect tampering.
#
# To verify: scripts/verify_goshi_sum.sh
# To regenerate: scripts/generate_goshi_sum.sh
#
EOF

echo "[1/4] Scanning Go source files..."

# Find all .go files, excluding vendor and .git
GO_FILES=$(find . -type f -name "*.go" ! -path "./vendor/*" ! -path "./.git/*" | sort)
FILE_COUNT=$(echo "$GO_FILES" | wc -l | tr -d ' ')

echo "  Found $FILE_COUNT Go source files"
echo ""

echo "[2/4] Computing SHA256 hashes..."

# Hash each file and append to goshi.sum
HASHED=0
while IFS= read -r file; do
    # Remove leading ./
    file_clean="${file#./}"
    
    # Compute SHA256 hash
    if command -v sha256sum &> /dev/null; then
        hash=$(sha256sum "$file" | awk '{print $1}')
    elif command -v shasum &> /dev/null; then
        hash=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        echo "  ERROR: No SHA256 tool found (sha256sum or shasum required)"
        exit 1
    fi
    
    # Append to goshi.sum
    echo "SHA256 $hash $file_clean" >> "$OUTPUT_FILE"
    
    HASHED=$((HASHED + 1))
    if (( HASHED % 10 == 0 )); then
        echo -ne "  Hashed $HASHED/$FILE_COUNT files\r"
    fi
done <<< "$GO_FILES"

echo "  Hashed $HASHED/$FILE_COUNT files    "
echo ""

echo "[3/4] Adding go.mod and go.sum..."

# Also hash go.mod and go.sum for dependency tracking
for dep_file in go.mod go.sum; do
    if [ -f "$dep_file" ]; then
        if command -v sha256sum &> /dev/null; then
            hash=$(sha256sum "$dep_file" | awk '{print $1}')
        else
            hash=$(shasum -a 256 "$dep_file" | awk '{print $1}')
        fi
        echo "SHA256 $hash $dep_file" >> "$OUTPUT_FILE"
        echo "  ✓ $dep_file"
    fi
done
echo ""

echo "[4/4] Finalizing manifest..."

# Add summary footer
cat >> "$OUTPUT_FILE" << EOF
#
# Summary:
# - Source files: $FILE_COUNT
# - Total hashes: $((FILE_COUNT + 2))
# - Generated: $BUILD_DATE
# - Commit: $GIT_COMMIT
#
EOF

TOTAL_SIZE=$(wc -c < "$OUTPUT_FILE")
echo "  ✓ Generated goshi.sum ($TOTAL_SIZE bytes)"
echo ""

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║                     GENERATION COMPLETE                        ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""
echo "Output: goshi.sum"
echo "Files hashed: $FILE_COUNT Go source files + 2 dependency files"
echo "Git state: $GIT_BRANCH @ $GIT_COMMIT${GIT_DIRTY}"
echo ""
echo "To verify integrity: scripts/verify_goshi_sum.sh"
