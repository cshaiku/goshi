package cli

import (
	"fmt"

	"github.com/cshaiku/goshi/internal/detect"
)

// PermissionHandler encapsulates permission request logic
// Extracted from runChat to improve Single Responsibility
type PermissionHandler struct {
	display    *DisplayConfig
	workingDir string
}

// NewPermissionHandler creates a permission handler
func NewPermissionHandler(workingDir string, display *DisplayConfig) *PermissionHandler {
	return &PermissionHandler{
		workingDir: workingDir,
		display:    display,
	}
}

// HandleDetected processes detected capabilities and requests permissions
// Returns true if all permissions granted, false if any denied
func (h *PermissionHandler) HandleDetected(detected []detect.Capability, session *ChatSession, systemPrompt string) bool {
	for _, cap := range detected {
		switch cap {
		case detect.CapabilityFSRead:
			if !session.HasPermission("FS_READ") {
				if !RequestFSReadPermission(h.workingDir) {
					h.refuseFSRead()
					session.DenyPermission("FS_READ")
					return false
				}
				session.GrantPermission("FS_READ")
				h.printStatus(systemPrompt, session.Permissions)
			}

		case detect.CapabilityFSWrite:
			if !session.HasPermission("FS_WRITE") {
				if !RequestFSWritePermission(h.workingDir) {
					h.refuseFSWrite()
					session.DenyPermission("FS_WRITE")
					return false
				}
				session.GrantPermission("FS_WRITE")
				h.printStatus(systemPrompt, session.Permissions)
			}
		}
	}
	return true
}

func (h *PermissionHandler) refuseFSRead() {
	fmt.Println("Filesystem access denied.\nPermission was not granted for this session.")
	fmt.Println("-----------------------------------------------------")
}

func (h *PermissionHandler) refuseFSWrite() {
	fmt.Println("Filesystem write access denied.\nPermission was not granted for this session.")
	fmt.Println("-----------------------------------------------------")
}

func (h *PermissionHandler) printStatus(systemPrompt string, perms *Permissions) {
	printStatus(systemPrompt, perms)
}
