package cli

import (
	"fmt"
	"os"

	"github.com/cshaiku/goshi/internal/detect"
	"github.com/cshaiku/goshi/internal/session"
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

// HandleDetected processes detected capabilities and requests permission from user
func (h *PermissionHandler) HandleDetected(detected []detect.Capability, sess *session.ChatSession, systemPrompt string) bool {
	for _, cap := range detected {
		switch cap {
		case detect.CapabilityFSRead:
			if !session.RequestFSReadPermission(h.workingDir) {
				return h.refuseFSRead(detected, sess)
			}
			sess.GrantPermission(string(cap))
		case detect.CapabilityFSWrite:
			if !session.RequestFSWritePermission(h.workingDir) {
				return h.refuseFSWrite(detected, sess)
			}
			sess.GrantPermission(string(cap))
		}
	}
	return true
}

func (h *PermissionHandler) refuseFSRead(detected []detect.Capability, sess *session.ChatSession) bool {
	fmt.Fprintf(os.Stderr, "%s\n", h.display.Colorize("Permission denied: FS_READ", ColorRed))
	sess.DenyPermission(string(detect.CapabilityFSRead))
	return false
}

func (h *PermissionHandler) refuseFSWrite(detected []detect.Capability, sess *session.ChatSession) bool {
	fmt.Fprintf(os.Stderr, "%s\n", h.display.Colorize("Permission denied: FS_WRITE", ColorRed))
	sess.DenyPermission(string(detect.CapabilityFSWrite))
	return false
}

func (h *PermissionHandler) getPermissionSummary(perms *session.Permissions) string {
	caps := []string{}
	if perms.FSRead {
		caps = append(caps, "FS_READ")
	}
	if perms.FSWrite {
		caps = append(caps, "FS_WRITE")
	}
	if len(caps) == 0 {
		return "STAGED (no permissions granted)"
	}
	return "ACTIVE: " + caps[0]
}
