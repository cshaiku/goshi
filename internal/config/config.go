package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
// Available Ollama models:
//   - qwen3:8b-q8_0 (recommended, 8.9GB, high quality)
//   - llama3:latest (4.7GB, good general purpose)
//   - llama3.1:8b (4.9GB, newer version)
//   - qwen2.5-coder:1.5b-base (986MB, code specialized)
func LoadDefaults() Config {
	return Config{
		LLM: LLMConfig{
			Model:          "qwen3:8b-q8_0",
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
		paths = append(paths, filepath.Join(wd, ".goshi.yaml"))
		paths = append(paths, filepath.Join(wd, "goshi.yaml"))
	}

	// 3. User home config
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".goshi", "config.yaml"))
		paths = append(paths, filepath.Join(home, ".goshi.yaml"))
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

	return nil
}

// Reset clears the cached config (useful for testing)
func Reset() {
	cachedConfig = nil
}
