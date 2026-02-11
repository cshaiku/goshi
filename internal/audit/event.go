package audit

import "time"

const SchemaVersion = "1"

type EventType string

type EventStatus string

const (
	EventTypePermission EventType = "permission"
	EventTypeTool       EventType = "tool"
	EventTypeSafety     EventType = "safety"
	EventTypeDiagnostic EventType = "diagnostic"
	EventTypeSession    EventType = "session"
	EventTypeMessage    EventType = "message"
	EventTypeResponse   EventType = "response"
)

const (
	StatusOK    EventStatus = "ok"
	StatusWarn  EventStatus = "warn"
	StatusError EventStatus = "error"
)

type Event struct {
	Timestamp time.Time      `json:"ts"`
	Type      EventType      `json:"type"`
	Action    string         `json:"action"`
	Status    EventStatus    `json:"status"`
	Message   string         `json:"message"`
	Cwd       string         `json:"cwd"`
	Details   map[string]any `json:"details,omitempty"`
	SessionID string         `json:"session_id"`
	Version   string         `json:"version"`
}
