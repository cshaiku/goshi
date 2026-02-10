# Goshi Write Capability - Fix Action Plan

## Overview
Goshi has the infrastructure for file writing (`fs.write` action exists) but it's blocked at the application layer in chat mode. This plan addresses 5 structural issues preventing the agent from executing write operations.

---

## Issue 1: Missing Enforcement in Tool Router
**File**: `internal/app/tool_router.go`
**Problem**: The `Handle()` method only enforces `CapFSRead` but has no guard for `CapFSWrite`
**Impact**: Even if CapFSWrite is granted, there's no capability enforcement
**Fix**: Add `case "fs.write"` to the capability switch statement to check `CapFSWrite`

**Status**: [x] COMPLETED

---

## Issue 2: No Write Detection Rules
**Files**: `internal/detect/rules_fs_read.go`, `internal/detect/engine.go`
**Problem**: Only `FSReadRules` exist; no detection rules for write operations
**Impact**: Chat mode cannot identify when user wants to perform writes
**Fix**:
  - Create `FSWriteRules` in a new file or extends rules_fs_read.go
  - Should detect verbs: write, create, save, update, edit, modify, patch, etc.
  - Should detect nouns: file, files, path, paths

**Status**: [x] COMPLETED

---

## Issue 3: No Write Permission Handshake
**File**: `internal/cli/chat.go`
**Problem**: Only `CapFSRead` is handled; no permission prompt or capability grant for writes
**Impact**: User cannot be asked for write permission; writes cannot be allowed
**Fix**:
  - Add `FSWrite` field to `Permissions` struct (line with `FSRead`)
  - Add detection handler for `CapabilityFSWrite` parallel to `FSRead` handling
  - Create `RequestFSWritePermission()` function (parallel to `RequestFSReadPermission()`)
  - Add direct action handler for write operations (similar to read/list regex handler)

**Status**: [x] COMPLETED

---

## Issue 4: Missing LLM Tool Instructions
**File**: `internal/llm/ollama/client.go`
**Problem**: `toolInstructions` constant only mentions `fs.list` and `fs.read`; LLM doesn't know about `fs.write`
**Impact**: LLM has no schema documentation for write operations
**Fix**: Update `toolInstructions` to include:
  - `fs.write` schema: `{"tool": "fs.write", "args": {"path": "...", "content": "..."}}`
  - Instructions that writes require explicit path and content

**Status**: [x] COMPLETED

---

## Issue 5: CLI Write Command Disconnected
**File**: `internal/cli/fs.go` (already implemented), integration into chat flow
**Problem**: Direct CLI `goshi fs write` works but isn't integrated with chat mode capability system
**Fix**: This is partially addressed by fixing Issues 1-4; ensure the tool_router properly delegates to dispatcher

**Status**: [x] COMPLETED (infrastructure now in place)

---

## Implementation Order
1. **Issue 1** (Tool Router) - Foundation for enforcement
2. **Issue 4** (LLM Instructions) - Inform the model
3. **Issue 2** (Detection Rules) - Identify user intent
4. **Issue 3** (Permission Handshake) - User consent
5. **Issue 5** (Full Integration) - Verify end-to-end

---

## Completion Summary

✅ **All 5 issues have been fixed**

### Changes Made:

1. **internal/app/tool_router.go**: Added `CapFSWrite` enforcement to the `Handle()` method
2. **internal/detect/engine.go**: Added `CapabilityFSWrite` constant
3. **internal/detect/rules_fs_read.go**: Created `FSWriteRules` with write-related verbs and nouns
4. **internal/llm/ollama/client.go**: Updated `toolInstructions` to document `fs.write` schema
5. **internal/cli/permissions.go**: 
   - Added `FSWrite` field to `Permissions` struct
   - Created `RequestFSWritePermission()` function
6. **internal/cli/chat.go**:
   - Updated `printStatus()` to show write permission status
   - Added `refuseFSWrite()` function
   - Enhanced capability detection to check both read and write rules
   - Added write capability handling with permission request

### How Write Operations Now Work:

1. User types a message containing write-related keywords (write, create, save, edit, etc.)
2. `FSWriteRules` detection identifies the write intent
3. System prompts user for write permission using `RequestFSWritePermission()`
4. Upon approval, `CapFSWrite` is granted to the capabilities
5. LLM receives the system prompt with write tool schema
6. LLM can request `fs.write` via tool calling interface
7. Tool router enforces `CapFSWrite` before executing
8. File is written (or error returned if permission denied)

