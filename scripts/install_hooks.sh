#!/bin/bash
# install_hooks.sh
# Installs Git hooks for automated checks

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HOOKS_DIR="$REPO_ROOT/.git/hooks"

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║              GOSHI GIT HOOKS INSTALLER                         ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Check if we're in a git repository
if [ ! -d "$REPO_ROOT/.git" ]; then
    echo "❌ ERROR: Not in a git repository"
    exit 1
fi

echo "Installing Git hooks to: $HOOKS_DIR"
echo ""

# Create hooks directory if it doesn't exist
mkdir -p "$HOOKS_DIR"

# Install pre-commit hook
echo "[1/2] Installing pre-commit hook..."
cp "$SCRIPT_DIR/hooks/pre-commit" "$HOOKS_DIR/pre-commit"
chmod +x "$HOOKS_DIR/pre-commit"
echo "  ✅ pre-commit hook installed"
echo ""

# Install pre-push hook
echo "[2/2] Installing pre-push hook..."
cp "$SCRIPT_DIR/hooks/pre-push" "$HOOKS_DIR/pre-push"
chmod +x "$HOOKS_DIR/pre-push"
echo "  ✅ pre-push hook installed"
echo ""

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║                 INSTALLATION COMPLETE                          ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""
echo "Git hooks installed successfully!"
echo ""
echo "Hooks installed:"
echo "  • pre-commit: Runs tests, vet, and format checks before commit"
echo "  • pre-push: Runs full tests and generates .goshi/goshi.manifest before push"
echo ""
echo "To skip hooks temporarily (not recommended):"
echo "  git commit --no-verify"
echo "  git push --no-verify"
echo ""
echo "To uninstall hooks:"
echo "  rm $HOOKS_DIR/pre-commit"
echo "  rm $HOOKS_DIR/pre-push"
