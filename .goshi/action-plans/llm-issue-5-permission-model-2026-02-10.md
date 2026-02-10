# Action Plan: Unified Permission Model

**Status:** Not Started  
**Priority:** P0 (Safety-critical)  
**Effort:** 1.5 days  
**Dependencies:** Issue #1 (Tool Registry)  
**Blocks:** Phase 3, 4

---

## Problem Statement

**Current Issue:** Permission enforcement is inconsistent across two checkpoints
- Pre-emptive regex detection in `chat.go` auto-grants permissions after user confirmation
- Separate per-tool capability checks in `tool_router.go` re-verify at execution time
- Ambiguous semantics: "Has user granted permission to fs.read forever?" or "Just this tool call?"
- No audit trail of permission decisions
- Permission state lives in `Permissions{}` struct with no history
- Cannot ask "Did user ever grant fs.write?" or "When did they grant it?"

**Risk:** Permission model unclear; users don't know if permissions are per-call or session-wide; cannot audit compliance.

---

## Solution Design

### 1. Permission Model: Session-Wide Grants

Establish clear semantics:
- Permissions are **session-scoped**: once granted, valid for entire chat session
- Cannot revoke permissions within session
- Each session starts with no permissions
- User can grant new permissions anytime
- Clear audit trail of all grants and denials

### 2. PermissionManager

Create `internal/cli/permission_model.go`:

```go
type PermissionType string

const (
    PermFSRead  PermissionType = "fs.read"
    PermFSWrite PermissionType = "fs.write"
)

type PermissionDecision struct {
    ID              string           // UUID for traceability
    Timestamp       time.Time
    Permission      PermissionType
    RequestedReason string           // Why tool needs it
    Granted         bool
    UserAction      string           // "approved" / "denied"
    ToolContext     string           // What tool wanted to do
}

type PermissionManager struct {
    decisions []PermissionDecision
    granted   map[PermissionType]bool
    mu        sync.RWMutex
}

// Request permission; returns true if granted
func (pm *PermissionManager) Request(
    perm PermissionType,
    reason string,
    toolContext string,
) bool {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    // Already granted?
    if pm.granted[perm] {
        pm.recordDecision(perm, reason, toolContext, true, "already_granted")
        return true
    }
    
    // Never asked; need user approval
    return false
}

// Ask user for permission; handle UI
func (pm *PermissionManager) RequestFromUser(
    perm PermissionType,
    reason string,
    toolContext string,
) bool {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    // Already granted?
    if pm.granted[perm] {
        pm.recordDecision(perm, reason, toolContext, true, "already_granted")
        return true
    }
    
    // Display permission request to user
    approved := askUserForPermission(perm, reason, toolContext)
    
    userAction := "denied"
    if approved {
        userAction = "approved"
        pm.granted[perm] = true
    }
    
    pm.recordDecision(perm, reason, toolContext, approved, userAction)
    return approved
}

func (pm *PermissionManager) recordDecision(
    perm PermissionType,
    reason string,
    toolContext string,
    granted bool,
    userAction string,
) {
    decision := PermissionDecision{
        ID:              uuid.New().String(),
        Timestamp:       time.Now(),
        Permission:      perm,
        RequestedReason: reason,
        Granted:         granted,
        UserAction:      userAction,
        ToolContext:     toolContext,
    }
    pm.decisions = append(pm.decisions, decision)
}

func (pm *PermissionManager) Has(perm PermissionType) bool {
    pm.mu.RLock()
    defer pm.mu.RUnlock()
    return pm.granted[perm]
}

func (pm *PermissionManager) AuditLog() []PermissionDecision {
    pm.mu.RLock()
    defer pm.mu.RUnlock()
    return append([]PermissionDecision{}, pm.decisions...)
}
```

### 3. Permission Declaration in Tool Registry

Tools declare their permission requirements:

```go
type ToolDefinition struct {
    ID              string
    Name            string
    Description     string
    RequiredPermission PermissionType
    PermissionReason   string  // Why this tool needs it
    Schema          JSONSchema
}

var FSReadTool = ToolDefinition{
    ID:                "fs.read",
    Name:              "Read File",
    Description:       "Read contents of a file from the repository",
    RequiredPermission: PermFSRead,
    PermissionReason:  "This tool reads files in your repository.",
    Schema:            // ...
}
```

### 4. Tool Router with Unified Permission Checking

Update `tool_router.go`:

```go
type ToolRouter struct {
    dispatcher *runtime.Dispatcher
    registry   *ToolRegistry
    permissions *PermissionManager  // NEW
}

func (r *ToolRouter) Handle(call ToolCall) Response {
    // 1. Look up tool
    toolDef, ok := r.registry.Get(call.Name)
    if !ok {
        return ErrorResponse("unknown tool")
    }
    
    // 2. Validate schema
    if err := r.registry.ValidateCall(call.Name, call.Args); err != nil {
        return ErrorResponse("invalid args: " + err.Error())
    }
    
    // 3. SINGLE permission check (no per-tool re-check)
    if !r.permissions.Has(toolDef.RequiredPermission) {
        return ErrorResponse(fmt.Sprintf(
            "permission denied: %s requires %s",
            call.Name,
            toolDef.RequiredPermission,
        ))
    }
    
    // 4. Execute
    out, err := r.dispatcher.Dispatch(call.Name, runtime.ActionInput(call.Args))
    if err != nil {
        return ErrorResponse(err.Error())
    }
    
    return SuccessResponse(out)
}
```

### 5. Chat Loop: Request Before LLM

New flow:
- User input arrives
- Detect what tool the user wants (heuristically or ask LLM)
- **Request permission BEFORE sending to LLM**
- Only then ask LLM to plan and execute

```go
func runChat(systemPrompt string) {
    // ...
    permMgr := cli.NewPermissionManager()
    
    for {
        fmt.Print("You: ")
        line, _ := reader.ReadString('\n')
        line = strings.TrimSpace(line)
        
        // Heuristic: Does user input mention reading/writing?
        intent := detectUserIntent(line)  // "read", "write", "list", "unknown"
        
        // Request permission if needed
        if intent == "read" && !permMgr.Has(PermFSRead) {
            approved := permMgr.RequestFromUser(
                PermFSRead,
                "The next operation may need to read files from your repository",
                fmt.Sprintf("User asked to: %s", line),
            )
            if !approved {
                fmt.Println("Filesystem read access denied for this session.")
                fmt.Println("-----------------------------------------------------")
                continue
            }
            fmt.Println("✓ Filesystem read access granted for this session.")
        }
        
        if intent == "write" && !permMgr.Has(PermFSWrite) {
            approved := permMgr.RequestFromUser(
                PermFSWrite,
                "The next operation may write files to your repository",
                fmt.Sprintf("User asked to: %s", line),
            )
            if !approved {
                fmt.Println("Filesystem write access denied for this session.")
                fmt.Println("-----------------------------------------------------")
                continue
            }
            fmt.Println("✓ Filesystem write access granted for this session.")
        }
        
        // Now send to LLM (permissions already granted)
        messages.Add(&UserMessage{Content: line}, "user_input", nil)
        
        // LLM now makes tool calls; tool_router just checks Have()
        // No more asking the user within the LLM loop
        
        // ... rest of chat loop ...
    }
}
```

### 6. Permission UI

```go
func askUserForPermission(
    perm PermissionType,
    reason string,
    toolContext string,
) bool {
    fmt.Println("\n" + strings.Repeat("-", 60))
    fmt.Printf("🔐 Permission Request\n")
    fmt.Printf("Reason: %s\n", reason)
    fmt.Printf("Context: %s\n", toolContext)
    fmt.Printf("\nGrant %s for this session? (yes/no): ", perm)
    
    reader := bufio.NewReader(os.Stdin)
    response, _ := reader.ReadString('\n')
    response = strings.ToLower(strings.TrimSpace(response))
    
    approved := response == "yes" || response == "y"
    
    if approved {
        fmt.Printf("✓ Granted: %s\n", perm)
    } else {
        fmt.Printf("✗ Denied: %s\n", perm)
    }
    fmt.Println(strings.Repeat("-", 60))
    
    return approved
}
```

### 7. Permission State Display

Add to `printStatus()`:

```go
func printSessionStatus(systemPrompt string, permMgr *PermissionManager) {
    // ... existing status ...
    fmt.Println("\n📋 Session Permissions:")
    
    if permMgr.Has(PermFSRead) {
        fmt.Println("  ✓ Filesystem read access granted")
    } else {
        fmt.Println("  ✗ Filesystem read access NOT granted")
    }
    
    if permMgr.Has(PermFSWrite) {
        fmt.Println("  ✓ Filesystem write access granted")
    } else {
        fmt.Println("  ✗ Filesystem write access NOT granted")
    }
    
    fmt.Println("-----------------------------------------------------")
}
```

### 8. Audit Log Export

Add command: `goshi session audit-log`

```go
// In CLI commands
func SessionAuditLog(permMgr *PermissionManager) {
    log := permMgr.AuditLog()
    
    for _, decision := range log {
        fmt.Printf(
            "[%s] %s: %s → %v (%s)\n",
            decision.Timestamp.Format("15:04:05"),
            decision.Permission,
            decision.RequestedReason,
            decision.Granted,
            decision.UserAction,
        )
    }
}
```

---

## Implementation Steps

1. **Create permission types** (`cli/permission_model.go`)
   - PermissionType enum
   - PermissionDecision struct
   - PermissionManager with Request/Has methods

2. **Integrate with tool registry** (`app/tools.go`)
   - Add RequiredPermission to ToolDefinition
   - Document why each tool needs permissions

3. **Update tool router** (`app/tool_router.go`)
   - Accept PermissionManager in constructor
   - Single permission check before dispatch
   - Clear error messages

4. **Refactor chat loop** (`cli/chat.go`)
   - Detect user intent (read/write/list/unknown)
   - Request permission from user BEFORE LLM
   - Display permission status at start
   - Remove old permission grant code

5. **Add audit logging** (`cli/session.go`)
   - Export permission decisions to JSON
   - Display audit log to user

6. **Tests** (`cli/permission_model_test.go`, `cli/chat_test.go`)
   - Test permission grant/deny flow
   - Test tool execution with/without permission
   - Test audit log recording
   - Test permission persistence across multiple tool calls

---

## Permission Flow Diagram

```
User Input
  ↓
Detect Intent (read/write/list)
  ↓
Permission Already Granted?
  ├─ YES → Proceed to LLM
  └─ NO → Ask User
        ├─ YES → Grant & Record & Proceed to LLM
        └─ NO → Deny & Record & Return to Prompt
  ↓
LLM Plans Tool Call
  ↓
Tool Router Checks Permission (should be true)
  ↓
Execute Tool
  ↓
Report Result
```

---

## Acceptance Criteria

- [ ] Permissions are session-scoped, not per-call
- [ ] Single permission check point in tool router
- [ ] User asked for permission BEFORE LLM gets involved
- [ ] All permission decisions logged with timestamp and context
- [ ] Tool definitions declare their permission requirements
- [ ] Permission audit log exportable and readable
- [ ] Clear UI showing current session permissions
- [ ] No permission re-asking on subsequent tool calls
- [ ] Test coverage for permission flows

---

## Security Properties

1. ✅ User always knows what permissions are being requested
2. ✅ Permissions cannot be implied or inferred; must be explicit
3. ✅ All permission decisions are auditable
4. ✅ Tool registry ensures consistent permission requirements
5. ✅ Single enforcement point prevents bypasses

---

## Notes

- Session-scoped permissions make sense for interactive CLI; user has full context
- Heuristic intent detection (read/write keywords) allows early request before LLM
- Can extend with scoping later (e.g., "only files matching *.md")
- Aligns with fail-closed semantics: deny by default, explicit grant required
