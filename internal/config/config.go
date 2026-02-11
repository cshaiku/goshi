package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LocalConfig holds local LLM server settings
type LocalConfig struct {
	URL  string `yaml:"url"`
	Port int    `yaml:"port"`
}

// LLMConfig holds LLM provider settings
type LLMConfig struct {
	Model          string      `yaml:"model"`
	Provider       string      `yaml:"provider"`
	Temperature    float32     `yaml:"temperature"`
	MaxTokens      int         `yaml:"max_tokens"`
	RequestTimeout int         `yaml:"request_timeout"`
	Local          LocalConfig `yaml:"local"`
}

// SafetyConfig holds safety and permission settings
type SafetyConfig struct {
	DryRunByDefault        bool `yaml:"dry_run_by_default"`
	AutoConfirmPermissions bool `yaml:"auto_confirm_permissions"`
	AutoBackupOnWrite      bool `yaml:"auto_backup_on_write"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level        string `yaml:"level"`
	OutputFormat string `yaml:"output_format"`
}

// AuditConfig holds audit log settings
type AuditConfig struct {
	Enabled            bool   `yaml:"enabled"`
	Dir                string `yaml:"dir"`
	RetentionDays      int    `yaml:"retention_days"`
	MaxSessions        int    `yaml:"max_sessions"`
	Redact             bool   `yaml:"redact"`
	ToolArgumentsStyle string `yaml:"tool_arguments_style"`
}

// BehaviorConfig holds behavioral settings
type BehaviorConfig struct {
	RepoRoot string `yaml:"repo_root"`
	CacheDir string `yaml:"cache_dir"`
}

// Config is the complete goshi configuration
type Config struct {
	LLM      LLMConfig      `yaml:"llm"`
	Safety   SafetyConfig   `yaml:"safety"`
	Logging  LoggingConfig  `yaml:"logging"`
	Audit    AuditConfig    `yaml:"audit"`
	Behavior BehaviorConfig `yaml:"behavior"`

	// Legacy CLI flags (for backward compatibility)
	Model       string
	LLMProvider string
	DryRun      bool
	Yes         bool
	JSON        bool
}

var cachedConfig *Config

// LoadDefaults returns a Config with safe defaults
// Available Ollama models (performance ranked for TUI):
//   - llama3.1:8b (RECOMMENDED for TUI - 4.9GB, ~0.19s cached response)
//   - llama3:latest (4.3GB, ~0.20s cached response)
//   - qwen2.5-coder:1.5b-base (986MB, code specialized)
//   - qwen3:8b-q8_0 (8.2GB, high quality but 38s+ on first load)
func LoadDefaults() Config {
	return Config{
		LLM: LLMConfig{
			Model:          "llama3.1:8b",
			Provider:       "ollama",
			Temperature:    0,
			MaxTokens:      4096,
			RequestTimeout: 60,
			Local: LocalConfig{
				URL:  "http://localhost",
				Port: 11434,
			},
		},
		Safety: SafetyConfig{
			DryRunByDefault:        true,
			AutoConfirmPermissions: false,
			AutoBackupOnWrite:      true,
		},
		Logging: LoggingConfig{
			Level:        "info",
			OutputFormat: "json",
		},
		Audit: AuditConfig{
			Enabled:            true,
			Dir:                ".goshi/audit",
			RetentionDays:      14,
			MaxSessions:        50,
			Redact:             true,
			ToolArgumentsStyle: "summaries",
		},
		Behavior: BehaviorConfig{
			RepoRoot: "",
			CacheDir: "",
		},
		DryRun: true,
		Yes:    false,
		JSON:   true,
	}
}

// configPaths returns candidate config file paths in priority order
func configPaths() []string {
	paths := []string{}

	// 1. Environment variable override
	if envPath := os.Getenv("GOSHI_CONFIG"); envPath != "" {
		paths = append(paths, envPath)
	}

	// 2. Repository-scoped config
	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(wd, "goshi.yaml"))
	}

	// 3. User home config
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".goshi", "config.yaml"))
	}

	// 4. System-wide config
	paths = append(paths, "/etc/goshi/config.yaml")

	return paths
}

// LoadYAML loads configuration from YAML file with fallback chain
func LoadYAML() (Config, error) {
	cfg := LoadDefaults()
	paths := configPaths()

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return cfg, fmt.Errorf("failed to read config at %s: %w", path, err)
		}

		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("failed to parse config at %s: %w", path, err)
		}

		// Found and loaded config
		return cfg, nil
	}

	// No config file found, return defaults
	return cfg, nil
}

// Load loads configuration with environment variable overrides
// This is the main entry point and uses caching
func Load() Config {
	if cachedConfig != nil {
		return *cachedConfig
	}

	cfg, _ := LoadYAML()

	// Apply environment variable overrides
	if model := os.Getenv("GOSHI_MODEL"); model != "" {
		cfg.Model = model
		cfg.LLM.Model = model
	}

	if provider := os.Getenv("GOSHI_LLM_PROVIDER"); provider != "" {
		cfg.LLMProvider = provider
		cfg.LLM.Provider = provider
	}

	if ollamaURL := os.Getenv("GOSHI_OLLAMA_URL"); ollamaURL != "" {
		cfg.LLM.Local.URL = ollamaURL
	}

	if ollamaPort := os.Getenv("GOSHI_OLLAMA_PORT"); ollamaPort != "" {
		fmt.Sscanf(ollamaPort, "%d", &cfg.LLM.Local.Port)
	}

	if auditEnabled := os.Getenv("GOSHI_AUDIT_ENABLED"); auditEnabled != "" {
		cfg.Audit.Enabled = parseBool(auditEnabled)
	}

	// Set defaults for legacy fields if not already set
	if cfg.Model == "" {
		cfg.Model = cfg.LLM.Model
	}
	if cfg.LLMProvider == "" {
		cfg.LLMProvider = cfg.LLM.Provider
	}

	cachedConfig = &cfg
	return cfg
}

func parseBool(value string) bool {
	switch strings.ToLower(value) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

// Validate checks configuration for errors
func (c *Config) Validate() error {
	if c.LLM.Model == "" {
		return errors.New("llm.model is required")
	}

	if c.LLM.Provider == "" {
		return errors.New("llm.provider is required")
	}

	if c.LLM.Temperature < 0 || c.LLM.Temperature > 2 {
		return fmt.Errorf("llm.temperature must be between 0 and 2, got %f", c.LLM.Temperature)
	}

	if c.LLM.MaxTokens <= 0 {
		return fmt.Errorf("llm.max_tokens must be positive, got %d", c.LLM.MaxTokens)
	}

	if c.LLM.RequestTimeout <= 0 {
		return fmt.Errorf("llm.request_timeout must be positive, got %d", c.LLM.RequestTimeout)
	}

	if c.LLM.Provider == "ollama" {
		if c.LLM.Local.URL == "" {
			return errors.New("llm.local.url is required for ollama provider")
		}
		if c.LLM.Local.Port <= 0 || c.LLM.Local.Port > 65535 {
			return fmt.Errorf("llm.local.port must be between 1 and 65535, got %d", c.LLM.Local.Port)
		}
	}

	if c.Logging.Level == "" ||
		(c.Logging.Level != "debug" &&
			c.Logging.Level != "info" &&
			c.Logging.Level != "warn" &&
			c.Logging.Level != "error") {
		return fmt.Errorf("logging.level must be debug, info, warn, or error, got %s", c.Logging.Level)
	}

	if c.Audit.ToolArgumentsStyle == "" {
		return errors.New("audit.tool_arguments_style is required")
	}

	switch c.Audit.ToolArgumentsStyle {
	case "full", "long", "short", "summaries":
		// valid
	default:
		return fmt.Errorf("audit.tool_arguments_style must be full, long, short, or summaries, got %s", c.Audit.ToolArgumentsStyle)
	}

	if c.Audit.RetentionDays < 0 {
		return fmt.Errorf("audit.retention_days must be >= 0, got %d", c.Audit.RetentionDays)
	}

	if c.Audit.MaxSessions < 0 {
		return fmt.Errorf("audit.max_sessions must be >= 0, got %d", c.Audit.MaxSessions)
	}

	return nil
}

// Reset clears the cached config (useful for testing)
func Reset() {
	cachedConfig = nil
}
