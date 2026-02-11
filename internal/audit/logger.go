package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type Config struct {
	Enabled            bool
	Dir                string
	RetentionDays      int
	MaxSessions        int
	Redact             bool
	ToolArgumentsStyle string
}

type Logger struct {
	cfg       Config
	dir       string
	filePath  string
	sessionID string
	file      *os.File
	mu        sync.Mutex
	enabled   bool
}

func NewLogger(cfg Config, repoRoot string) (*Logger, error) {
	if !cfg.Enabled {
		return &Logger{cfg: cfg, enabled: false}, nil
	}

	dir, err := resolveDir(cfg.Dir, repoRoot)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit dir: %w", err)
	}

	if err := cleanupOldSessions(dir, cfg.RetentionDays, cfg.MaxSessions); err != nil {
		return nil, err
	}

	sessionID := newSessionID()
	filePath := filepath.Join(dir, fmt.Sprintf("session-%s.jsonl", sessionID))
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}

	logger := &Logger{
		cfg:       cfg,
		dir:       dir,
		filePath:  filePath,
		sessionID: sessionID,
		file:      file,
		enabled:   true,
	}

	return logger, nil
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	return l.file.Close()
}

func (l *Logger) SessionID() string {
	return l.sessionID
}

func (l *Logger) FilePath() string {
	return l.filePath
}

func (l *Logger) LogEvent(event Event) {
	if !l.enabled {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.Version == "" {
		event.Version = SchemaVersion
	}
	if event.SessionID == "" {
		event.SessionID = l.sessionID
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	_, _ = l.file.Write(append(data, '\n'))
}

func (l *Logger) LogPermission(action string, capability string, reason string, cwd string) {
	l.LogEvent(Event{
		Type:    EventTypePermission,
		Action:  action,
		Status:  StatusOK,
		Message: fmt.Sprintf("%s %s (%s)", action, capability, reason),
		Cwd:     cwd,
		Details: map[string]any{
			"capability": capability,
			"reason":     reason,
		},
	})
}

func (l *Logger) LogTool(name string, status EventStatus, message string, args map[string]any, cwd string) {
	l.LogEvent(Event{
		Type:    EventTypeTool,
		Action:  name,
		Status:  status,
		Message: message,
		Cwd:     cwd,
		Details: map[string]any{
			"args": FormatToolArgs(args, l.cfg.ToolArgumentsStyle, l.cfg.Redact),
		},
	})
}

func (l *Logger) LogDiagnostic(code string, status EventStatus, message string, cwd string) {
	l.LogEvent(Event{
		Type:    EventTypeDiagnostic,
		Action:  code,
		Status:  status,
		Message: message,
		Cwd:     cwd,
	})
}

func (l *Logger) LogSafety(code string, status EventStatus, message string, cwd string) {
	l.LogEvent(Event{
		Type:    EventTypeSafety,
		Action:  code,
		Status:  status,
		Message: message,
		Cwd:     cwd,
	})
}

func (l *Logger) LogSession(action string, message string, cwd string) {
	l.LogEvent(Event{
		Type:    EventTypeSession,
		Action:  action,
		Status:  StatusOK,
		Message: message,
		Cwd:     cwd,
	})
}

func resolveDir(dir string, repoRoot string) (string, error) {
	if dir == "" {
		dir = ".goshi/audit"
	}

	if filepath.IsAbs(dir) {
		return dir, nil
	}

	if repoRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to resolve audit dir: %w", err)
		}
		repoRoot = cwd
	}

	return filepath.Join(repoRoot, dir), nil
}

func cleanupOldSessions(dir string, retentionDays int, maxSessions int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	now := time.Now()
	type fileEntry struct {
		name string
		info os.FileInfo
	}
	var files []fileEntry

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !stringsHasPrefix(name, "session-") || !stringsHasSuffix(name, ".jsonl") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, fileEntry{name: name, info: info})
	}

	if retentionDays > 0 {
		cutoff := now.Add(-time.Duration(retentionDays) * 24 * time.Hour)
		for _, file := range files {
			if file.info.ModTime().Before(cutoff) {
				_ = os.Remove(filepath.Join(dir, file.name))
			}
		}
	}

	if maxSessions > 0 {
		sort.Slice(files, func(i, j int) bool {
			return files[i].info.ModTime().After(files[j].info.ModTime())
		})
		if len(files) > maxSessions {
			for _, file := range files[maxSessions:] {
				_ = os.Remove(filepath.Join(dir, file.name))
			}
		}
	}

	return nil
}

func newSessionID() string {
	stamp := time.Now().UTC().Format("20060102-150405.000")
	return fmt.Sprintf("%s-%d", stamp, os.Getpid())
}

func stringsHasPrefix(value string, prefix string) bool {
	if len(prefix) > len(value) {
		return false
	}
	return value[:len(prefix)] == prefix
}

func stringsHasSuffix(value string, suffix string) bool {
	if len(suffix) > len(value) {
		return false
	}
	return value[len(value)-len(suffix):] == suffix
}
