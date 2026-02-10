package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/manifoldco/promptui"
)

// PermissionEntry represents an audit log entry for a permission decision
type PermissionEntry struct {
	Capability string    `json:"capability"` // e.g., "FS_READ", "FS_WRITE"
	Action     string    `json:"action"`     // "GRANT", "DENY", "REVOKE"
	Timestamp  time.Time `json:"timestamp"`
	Reason     string    `json:"reason"` // User decision or auto-confirm reason
	RequestCwd string    `json:"request_cwd"`
}

// Permissions represents the current session permissions with audit trail
type Permissions struct {
	FSRead   bool
	FSWrite  bool
	AuditLog []PermissionEntry // Complete decision history
}

// Grant records a permission grant in the audit log
func (p *Permissions) Grant(capability string, cwd string) {
	entry := PermissionEntry{
		Capability: capability,
		Action:     "GRANT",
		Timestamp:  time.Now(),
		Reason:     "user-approved",
		RequestCwd: cwd,
	}

	switch capability {
	case "FS_READ":
		p.FSRead = true
	case "FS_WRITE":
		p.FSWrite = true
	}

	p.AuditLog = append(p.AuditLog, entry)
}

// Deny records a permission denial in the audit log
func (p *Permissions) Deny(capability string, cwd string) {
	entry := PermissionEntry{
		Capability: capability,
		Action:     "DENY",
		Timestamp:  time.Now(),
		Reason:     "user-denied",
		RequestCwd: cwd,
	}
	p.AuditLog = append(p.AuditLog, entry)
}

// AutoConfirm grants a permission via auto-confirm mechanism
func (p *Permissions) AutoConfirm(capability string, cwd string) {
	entry := PermissionEntry{
		Capability: capability,
		Action:     "GRANT",
		Timestamp:  time.Now(),
		Reason:     "auto-confirm-enabled",
		RequestCwd: cwd,
	}

	switch capability {
	case "FS_READ":
		p.FSRead = true
	case "FS_WRITE":
		p.FSWrite = true
	}

	p.AuditLog = append(p.AuditLog, entry)
}

// HasPermission checks if a capability is currently granted
func (p *Permissions) HasPermission(capability string) bool {
	switch capability {
	case "FS_READ":
		return p.FSRead
	case "FS_WRITE":
		return p.FSWrite
	default:
		return false
	}
}

// GetAuditTrail returns a formatted audit trail for logging
func (p *Permissions) GetAuditTrail() string {
	if len(p.AuditLog) == 0 {
		return ""
	}

	trail := "Permission Audit Trail:\n"
	for _, entry := range p.AuditLog {
		trail += fmt.Sprintf("  [%s] %s %s (%s) in %s\n",
			entry.Timestamp.Format("15:04:05"),
			entry.Action,
			entry.Capability,
			entry.Reason,
			entry.RequestCwd,
		)
	}
	return trail
}

func RequestFSReadPermission(cwd string) bool {
	cfg := GetConfig()
	if cfg.Safety.AutoConfirmPermissions {
		return true
	}
	items := []string{
		"Allow read-only access (this session)",
		"Deny",
		"Abort request",
	}

	prompt := promptui.Select{
		Label: fmt.Sprintf(
			"Goshi requires permission to read files in:\n  %s\n\nWhat would you like to do?",
			cwd,
		),
		Items: items,
	}

	i, _, err := prompt.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "permission prompt cancelled")
		return false
	}

	switch i {
	case 0:
		return true
	case 1:
		return false
	default:
		return false
	}
}

func RequestFSWritePermission(cwd string) bool {
	cfg := GetConfig()
	if cfg.Safety.AutoConfirmPermissions {
		return true
	}
	items := []string{
		"Allow write access (this session)",
		"Deny",
		"Abort request",
	}

	prompt := promptui.Select{
		Label: fmt.Sprintf(
			"Goshi requests permission to write files in:\n  %s\n\nWhat would you like to do?",
			cwd,
		),
		Items: items,
	}

	i, _, err := prompt.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "permission prompt cancelled")
		return false
	}

	switch i {
	case 0:
		return true
	case 1:
		return false
	default:
		return false
	}
}
