#!/usr/bin/env bash
# Build a native developer binary and atomically install it outside the checkout.
# PREFIX defaults to the user-local POSIX prefix. CI/cross-builds use --compile-only.
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tool_name='goshi'
package='.'
compile_only=0
case "${1:-}" in
    --compile-only) compile_only=1 ;;
    '') ;;
    *) printf 'Usage: %s [--compile-only]\n' "$0" >&2; exit 2 ;;
esac
[[ $# -le 1 ]] || exit 2
go_bin="${GO:-go}"
prefix="${PREFIX:-$HOME/.local}"
case "$prefix" in /*) ;; *) printf 'PREFIX must be absolute: %s\n' "$prefix" >&2; exit 2 ;; esac
cd "$repo_root"
if [[ "$compile_only" == 0 ]]; then
    [[ "$("$go_bin" env GOOS)/$("$go_bin" env GOARCH)" == "$("$go_bin" env GOHOSTOS)/$("$go_bin" env GOHOSTARCH)" ]] || {
        printf 'Refusing to install a cross-compiled binary; use make compile.\n' >&2; exit 1;
    }
fi
mkdir -p .build
"$go_bin" build -trimpath -o ".build/$tool_name" "$package"
if [[ "$compile_only" == 1 ]]; then
    printf 'Built only: %s/.build/%s\n' "$repo_root" "$tool_name"
    exit 0
fi
# Stage a complete payload before changing an installed command. Retain each
# previous install for rollback; no source checkout or user configuration is removed.
mkdir -p "$prefix/bin" "$prefix/share/$tool_name/installs"
stage="$(mktemp -d "$prefix/share/$tool_name/installs/$(date -u +%Y%m%dT%H%M%SZ).XXXXXX")"
mkdir -p "$stage/bin" "$stage/share/$tool_name"
install -m 755 ".build/$tool_name" "$stage/bin/$tool_name"
install -m 644 goshi.self.model.yaml "$stage/share/goshi/goshi.self.model.yaml"
{
    printf 'tool=%s\nsource=%s\ncommit=%s\ninstalled_at=%s\n' "$tool_name" "$repo_root" "$(git rev-parse HEAD)" "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    printf 'sha256=%s\n' "$(shasum -a 256 "$stage/bin/$tool_name" | cut -d ' ' -f 1)"
    printf 'go=%s\n' "$("$go_bin" version)"
    printf 'working_tree_changes:\n'
    git status --short
} > "$stage/install-info.txt"
# Binary is a real installed file, not a symlink into this repo. Back up any old
# binary, including a legacy symlink, before atomic replacement in the same dir.
if [[ -e "$prefix/bin/$tool_name" || -L "$prefix/bin/$tool_name" ]]; then
    cp -Pp "$prefix/bin/$tool_name" "$stage/previous-command"
fi
binary_tmp="$(mktemp "$prefix/bin/.$tool_name.XXXXXX")"
trap 'rm -f "$binary_tmp"' EXIT
install -m 755 "$stage/bin/$tool_name" "$binary_tmp"
if [[ -e "$prefix/share/goshi/goshi.self.model.yaml" ]]; then
    cp -p "$prefix/share/goshi/goshi.self.model.yaml" "$stage/previous-self-model.yaml"
fi
install -m 644 "$stage/share/goshi/goshi.self.model.yaml" "$prefix/share/goshi/goshi.self.model.yaml"
mv -f "$binary_tmp" "$prefix/bin/$tool_name"
cp "$stage/install-info.txt" "$prefix/share/$tool_name/install-info.txt"
printf 'Installed: %s/bin/%s\nReceipt: %s/share/%s/install-info.txt\n' "$prefix" "$tool_name" "$prefix" "$tool_name"
