package audit

import (
	"os"
	"testing"
)

func TestFormatToolArgsSummariesRedacts(t *testing.T) {
	args := map[string]any{
		"token": "secret-value",
		"path":  "dir/file.txt",
		"list":  []any{1, 2, 3},
	}

	out := FormatToolArgs(args, "summaries", true)

	if out["token"] != "***redacted***" {
		t.Fatalf("expected token to be redacted")
	}
	if out["path"] != "dir/file.txt" {
		t.Fatalf("expected path to be preserved in summaries")
	}
	if out["list"] != "array[3]" {
		t.Fatalf("expected list summary, got %v", out["list"])
	}
}

func TestLoggerWritesEvents(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{
		Enabled:            true,
		Dir:                tmpDir,
		RetentionDays:      0,
		MaxSessions:        0,
		Redact:             true,
		ToolArgumentsStyle: "summaries",
	}

	logger, err := NewLogger(cfg, "")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	logger.LogSession("START", "session started", "/tmp")

	if _, err := os.Stat(logger.FilePath()); err != nil {
		t.Fatalf("expected audit file to exist: %v", err)
	}

	events, err := ReadEvents(logger.FilePath(), Filter{})
	if err != nil {
		t.Fatalf("failed to read events: %v", err)
	}
	if len(events) == 0 {
		t.Fatalf("expected at least 1 event")
	}
	if events[0].Type != EventTypeSession {
		t.Fatalf("expected session event, got %s", events[0].Type)
	}
}
