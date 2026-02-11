#!/bin/bash
# generate_goshi_manifest.sh
# Generates a source tarball and integrity manifest for Go files

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
MANIFEST_DIR="$REPO_ROOT/.goshi"
MANIFEST_PATH="$MANIFEST_DIR/goshi.manifest"
TARBALL_PATH="$MANIFEST_DIR/goshi.source.tar.gz"

if [ ! -d "$REPO_ROOT/.git" ]; then
    echo "ERROR: Not in a git repository"
    exit 1
fi

mkdir -p "$MANIFEST_DIR"

if git rev-parse --git-dir > /dev/null 2>&1; then
    GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
    GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
    GIT_TAG=$(git describe --tags --exact-match 2>/dev/null || echo "none")
    if git diff --quiet; then
        GIT_DIRTY="false"
    else
        GIT_DIRTY="true"
    fi
else
    GIT_COMMIT="no-git"
    GIT_BRANCH="no-git"
    GIT_TAG="none"
    GIT_DIRTY="false"
fi

BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION=$(go version 2>/dev/null | awk '{print $3}' || echo "unknown")

PYTHON_BIN=""
if command -v python3 >/dev/null 2>&1; then
    PYTHON_BIN="python3"
elif command -v python >/dev/null 2>&1; then
    PYTHON_BIN="python"
else
    echo "ERROR: python3 or python is required"
    exit 1
fi

REPO_ROOT="$REPO_ROOT" \
MANIFEST_PATH="$MANIFEST_PATH" \
TARBALL_PATH="$TARBALL_PATH" \
BUILD_DATE="$BUILD_DATE" \
GIT_COMMIT="$GIT_COMMIT" \
GIT_BRANCH="$GIT_BRANCH" \
GIT_TAG="$GIT_TAG" \
GIT_DIRTY="$GIT_DIRTY" \
GO_VERSION="$GO_VERSION" \
"$PYTHON_BIN" - <<'PY'
import datetime
import gzip
import hashlib
import os
import tarfile

repo_root = os.environ["REPO_ROOT"]
manifest_path = os.environ["MANIFEST_PATH"]
tarball_path = os.environ["TARBALL_PATH"]

build_date = os.environ.get("BUILD_DATE", "unknown")
git_commit = os.environ.get("GIT_COMMIT", "unknown")
git_branch = os.environ.get("GIT_BRANCH", "unknown")
git_tag = os.environ.get("GIT_TAG", "none")
git_dirty = os.environ.get("GIT_DIRTY", "false")
go_version = os.environ.get("GO_VERSION", "unknown")

skip_dirs = {".git", "vendor", ".goshi"}

file_paths = []
for root, dirs, files in os.walk(repo_root):
    dirs[:] = [d for d in dirs if d not in skip_dirs]
    for name in files:
        if not name.endswith(".go"):
            continue
        abs_path = os.path.join(root, name)
        rel_path = os.path.relpath(abs_path, repo_root)
        file_paths.append(rel_path)

file_paths.sort()

def sha256_file(path: str) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as f:
        while True:
            chunk = f.read(1024 * 1024)
            if not chunk:
                break
            h.update(chunk)
    return h.hexdigest()

entries = []
for rel_path in file_paths:
    abs_path = os.path.join(repo_root, rel_path)
    st = os.stat(abs_path)
    mode = st.st_mode & 0o777
    mtime = datetime.datetime.fromtimestamp(st.st_mtime, tz=datetime.timezone.utc)
    entries.append({
        "path": rel_path,
        "sha256": sha256_file(abs_path),
        "size": st.st_size,
        "mode": f"{mode:04o}",
        "mtime": mtime.strftime("%Y-%m-%dT%H:%M:%SZ"),
    })

with open(tarball_path, "wb") as raw:
    with gzip.GzipFile(fileobj=raw, mode="wb", mtime=0) as gz:
        with tarfile.open(fileobj=gz, mode="w", format=tarfile.PAX_FORMAT) as tar:
            for rel_path in file_paths:
                abs_path = os.path.join(repo_root, rel_path)
                st = os.stat(abs_path)
                tar_info = tarfile.TarInfo(name=rel_path)
                tar_info.size = st.st_size
                tar_info.mode = st.st_mode & 0o777
                tar_info.mtime = 0
                tar_info.uid = 0
                tar_info.gid = 0
                tar_info.uname = ""
                tar_info.gname = ""
                with open(abs_path, "rb") as f:
                    tar.addfile(tar_info, f)


tarball_size = os.path.getsize(tarball_path)
tarball_hash = sha256_file(tarball_path)

with open(manifest_path, "w", encoding="utf-8") as f:
    f.write("# goshi.manifest - Source Integrity Manifest\n")
    f.write("# Version: 1\n")
    f.write(f"# Generated: {build_date}\n")
    f.write(f"# Git Commit: {git_commit}\n")
    f.write(f"# Git Branch: {git_branch}\n")
    f.write(f"# Git Tag: {git_tag}\n")
    f.write(f"# Git Dirty: {git_dirty}\n")
    f.write(f"# Go Version: {go_version}\n")
    f.write(f"# Source Tarball: {os.path.relpath(tarball_path, repo_root)}\n")
    f.write("#\n")
    f.write("# Format:\n")
    f.write("#   TARBALL <sha256> <size> <path>\n")
    f.write("#   FILE <sha256> <size> <mode> <mtime> <path>\n")
    f.write("#\n")
    f.write("VERSION 1\n")
    f.write(f"TARBALL {tarball_hash} {tarball_size} {os.path.relpath(tarball_path, repo_root)}\n")
    for entry in entries:
        f.write(
            "FILE {sha256} {size} {mode} {mtime} {path}\n".format(**entry)
        )
    f.write("#\n")
    f.write(f"# File Count: {len(entries)}\n")
    f.write(f"# Generated: {build_date}\n")
PY

echo "Generated source tarball: $TARBALL_PATH"
echo "Generated manifest: $MANIFEST_PATH"
