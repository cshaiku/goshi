package session

import (
	"testing"
	"time"
)

func TestPermissions_Grant(t *testing.T) {
	perms := &Permissions{
		AuditLog: []PermissionEntry{},
	}

	perms.Grant("FS_READ", "/home/user")

	if !perms.FSRead {
		t.Error("FSRead should be true after Grant")
	}

	if len(perms.AuditLog) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(perms.AuditLog))
	}

	entry := perms.AuditLog[0]
	if entry.Action != "GRANT" {
		t.Errorf("expected action GRANT, got %s", entry.Action)
	}
	if entry.Capability != "FS_READ" {
		t.Errorf("expected capability FS_READ, got %s", entry.Capability)
	}
	if entry.Reason != "user-approved" {
		t.Errorf("expected reason user-approved, got %s", entry.Reason)
	}
}

func TestPermissions_Deny(t *testing.T) {
	perms := &Permissions{
		AuditLog: []PermissionEntry{},
	}

	perms.Deny("FS_WRITE", "/home/user")

	if perms.FSWrite {
		t.Error("FSWrite should be false after Deny")
	}

	if len(perms.AuditLog) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(perms.AuditLog))
	}

	entry := perms.AuditLog[0]
	if entry.Action != "DENY" {
		t.Errorf("expected action DENY, got %s", entry.Action)
	}
}

func TestPermissions_AutoConfirm(t *testing.T) {
	perms := &Permissions{
		AuditLog: []PermissionEntry{},
	}

	perms.AutoConfirm("FS_READ", "/home/user")

	if !perms.FSRead {
		t.Error("FSRead should be true after AutoConfirm")
	}

	if len(perms.AuditLog) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(perms.AuditLog))
	}

	entry := perms.AuditLog[0]
	if entry.Reason != "auto-confirm-enabled" {
		t.Errorf("expected reason auto-confirm-enabled, got %s", entry.Reason)
	}
}

func TestPermissions_HasPermission(t *testing.T) {
	perms := &Permissions{
		FSRead:   true,
		FSWrite:  false,
		AuditLog: []PermissionEntry{},
	}

	if !perms.HasPermission("FS_READ") {
		t.Error("should have FS_READ permission")
	}

	if perms.HasPermission("FS_WRITE") {
		t.Error("should not have FS_WRITE permission")
	}

	if perms.HasPermission("UNKNOWN") {
		t.Error("should not have UNKNOWN permission")
	}
}

func TestPermissions_AuditTrail(t *testing.T) {
	perms := &Permissions{
		AuditLog: []PermissionEntry{},
	}

	perms.Grant("FS_READ", "/home/user")
	time.Sleep(10 * time.Millisecond)
	perms.Deny("FS_WRITE", "/home/user")

	trail := perms.GetAuditTrail()

	if trail == "" {
		t.Error("audit trail should not be empty")
	}

	if len(perms.AuditLog) != 2 {
		t.Errorf("expected 2 audit entries, got %d", len(perms.AuditLog))
	}

	// Verify entries are in correct order
	if perms.AuditLog[0].Action != "GRANT" {
		t.Error("first entry should be GRANT")
	}
	if perms.AuditLog[1].Action != "DENY" {
		t.Error("second entry should be DENY")
	}

	// Verify timestamps are recorded
	if perms.AuditLog[0].Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
}

func TestPermissions_MultiplePermissions(t *testing.T) {
	perms := &Permissions{
		AuditLog: []PermissionEntry{},
	}

	perms.Grant("FS_READ", "/home")
	perms.Grant("FS_WRITE", "/home")

	if !perms.FSRead || !perms.FSWrite {
		t.Error("both permissions should be granted")
	}

	if len(perms.AuditLog) != 2 {
		t.Fatalf("expected 2 audit entries, got %d", len(perms.AuditLog))
	}

	// Verify each entry has correct capability
	if perms.AuditLog[0].Capability != "FS_READ" {
		t.Error("first entry should be FS_READ")
	}
	if perms.AuditLog[1].Capability != "FS_WRITE" {
		t.Error("second entry should be FS_WRITE")
	}
}

func TestPermissions_EmptyAuditTrail(t *testing.T) {
	perms := &Permissions{
		AuditLog: []PermissionEntry{},
	}

	trail := perms.GetAuditTrail()

	if trail != "" {
		t.Error("audit trail should be empty when no entries exist")
	}
}

func TestPermissionEntry_Fields(t *testing.T) {
	entry := PermissionEntry{
		Capability: "FS_READ",
		Action:     "GRANT",
		Timestamp:  time.Now(),
		Reason:     "test-reason",
		RequestCwd: "/tmp",
	}

	if entry.Capability != "FS_READ" {
		t.Error("capability not set correctly")
	}
	if entry.Action != "GRANT" {
		t.Error("action not set correctly")
	}
	if entry.RequestCwd != "/tmp" {
		t.Error("request cwd not set correctly")
	}
}
