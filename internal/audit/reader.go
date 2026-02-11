package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Filter struct {
	Types  map[EventType]bool
	Status map[EventStatus]bool
	Since  time.Time
	Until  time.Time
	Limit  int
}

func ReadEvents(path string, filter Filter) ([]Event, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var events []Event

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var event Event
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}
		if !passesFilter(event, filter) {
			continue
		}
		events = append(events, event)
		if filter.Limit > 0 && len(events) >= filter.Limit {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read audit log: %w", err)
	}

	return events, nil
}

func LatestSessionFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read audit dir: %w", err)
	}

	type fileEntry struct {
		name string
		mod  time.Time
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
		files = append(files, fileEntry{name: name, mod: info.ModTime()})
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no audit sessions found")
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].mod.After(files[j].mod)
	})

	return filepath.Join(dir, files[0].name), nil
}

func passesFilter(event Event, filter Filter) bool {
	if len(filter.Types) > 0 {
		if !filter.Types[event.Type] {
			return false
		}
	}
	if len(filter.Status) > 0 {
		if !filter.Status[event.Status] {
			return false
		}
	}
	if !filter.Since.IsZero() && event.Timestamp.Before(filter.Since) {
		return false
	}
	if !filter.Until.IsZero() && event.Timestamp.After(filter.Until) {
		return false
	}
	return true
}
