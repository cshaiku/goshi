package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cshaiku/goshi/internal/audit"
	"github.com/cshaiku/goshi/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newAuditCommand() *cobra.Command {
	var format string
	var session string
	var since string
	var until string
	var limit int
	var types string
	var status string
	var unsafe bool

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "View audit logs",
		Long: `View audit logs for the current repository.

By default, shows the latest session. Use --session to select a specific session
or --format=json for structured output.

EXAMPLES:
  goshi audit
  goshi audit --format=json --limit=200
  goshi audit --since=1h --type=tool
  goshi audit --session=session-20260210-153000.000-1234`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Load()
			repoRoot := cfg.Behavior.RepoRoot
			if repoRoot == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				repoRoot = cwd
			}

			auditDir := cfg.Audit.Dir
			if auditDir == "" {
				auditDir = ".goshi/audit"
			}
			if !filepath.IsAbs(auditDir) {
				auditDir = filepath.Join(repoRoot, auditDir)
			}

			filePath := ""
			if session != "" {
				if strings.HasSuffix(session, ".jsonl") {
					filePath = filepath.Join(auditDir, session)
				} else {
					filePath = filepath.Join(auditDir, fmt.Sprintf("%s.jsonl", session))
				}
			} else {
				latest, err := audit.LatestSessionFile(auditDir)
				if err != nil {
					return err
				}
				filePath = latest
			}

			filter := audit.Filter{}
			if limit > 0 {
				filter.Limit = limit
			}
			if since != "" {
				parsed, err := parseTimeOrDuration(since)
				if err != nil {
					return err
				}
				filter.Since = parsed
			}
			if until != "" {
				parsed, err := time.Parse(time.RFC3339, until)
				if err != nil {
					return fmt.Errorf("invalid --until time: %w", err)
				}
				filter.Until = parsed
			}
			if types != "" {
				filter.Types = make(map[audit.EventType]bool)
				for _, item := range strings.Split(types, ",") {
					filter.Types[audit.EventType(strings.TrimSpace(item))] = true
				}
			}
			if status != "" {
				filter.Status = make(map[audit.EventStatus]bool)
				for _, item := range strings.Split(status, ",") {
					filter.Status[audit.EventStatus(strings.TrimSpace(item))] = true
				}
			}

			events, err := audit.ReadEvents(filePath, filter)
			if err != nil {
				return err
			}

			if unsafe {
				// No redaction changes are applied here; logs are already redacted by writer.
				// This flag is reserved for future use and alignment with config.
			}

			switch format {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(events)
			case "yaml":
				data, err := yaml.Marshal(events)
				if err != nil {
					return err
				}
				fmt.Print(string(data))
				return nil
			case "human", "":
				for _, event := range events {
					fmt.Printf("[%s] %-10s %-8s %s\n",
						event.Timestamp.Format("15:04:05"),
						event.Type,
						event.Status,
						event.Message,
					)
				}
				return nil
			default:
				return fmt.Errorf("unknown format: %s (use human, json, or yaml)", format)
			}
		},
	}

	cmd.Flags().StringVar(&format, "format", "human", "Output format: human, json, or yaml")
	cmd.Flags().StringVar(&session, "session", "", "Session ID or filename (default: latest)")
	cmd.Flags().StringVar(&since, "since", "", "Show events since duration or RFC3339 time (e.g., 1h or 2026-02-10T12:00:00Z)")
	cmd.Flags().StringVar(&until, "until", "", "Show events until RFC3339 time (e.g., 2026-02-10T12:30:00Z)")
	cmd.Flags().IntVar(&limit, "limit", 200, "Maximum number of events to show")
	cmd.Flags().StringVar(&types, "type", "", "Comma-separated event types (permission, tool, safety, diagnostic, session)")
	cmd.Flags().StringVar(&status, "status", "", "Comma-separated status filters (ok, warn, error)")
	cmd.Flags().BoolVar(&unsafe, "unsafe", false, "Reserved: allow unredacted output if available")
	return cmd
}

func parseTimeOrDuration(value string) (time.Time, error) {
	if strings.HasSuffix(value, "h") || strings.HasSuffix(value, "m") || strings.HasSuffix(value, "s") {
		duration, err := time.ParseDuration(value)
		if err == nil {
			return time.Now().Add(-duration), nil
		}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --since value: %w", err)
	}
	return parsed, nil
}
