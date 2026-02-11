package config

import (
	"os"
	"testing"
)

// TestLoadDefaults verifies that LoadDefaults returns a valid default configuration
func TestLoadDefaults(t *testing.T) {
	cfg := LoadDefaults()

	if cfg.LLM.Model != "qwen3:8b-q8_0" {
		t.Errorf("expected default model to be qwen3:8b-q8_0, got %s", cfg.LLM.Model)
	}

	if cfg.LLM.Provider != "ollama" {
		t.Errorf("expected default provider to be ollama, got %s", cfg.LLM.Provider)
	}

	if cfg.LLM.Temperature != 0 {
		t.Errorf("expected default temperature 0, got %f", cfg.LLM.Temperature)
	}

	if cfg.LLM.MaxTokens != 4096 {
		t.Errorf("expected default max_tokens 4096, got %d", cfg.LLM.MaxTokens)
	}

	if cfg.LLM.Local.Port != 11434 {
		t.Errorf("expected default ollama port 11434, got %d", cfg.LLM.Local.Port)
	}

	if cfg.Safety.DryRunByDefault != true {
		t.Errorf("expected dry_run_by_default to be true")
	}

	if cfg.Safety.AutoConfirmPermissions != false {
		t.Errorf("expected auto_confirm_permissions to be false")
	}
}

// TestValidateSuccess tests validation of a valid config
func TestValidateSuccess(t *testing.T) {
	cfg := LoadDefaults()
	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected valid config to pass validation, got error: %v", err)
	}
}

// TestValidateMissingModel tests that validation fails without a model
func TestValidateMissingModel(t *testing.T) {
	cfg := LoadDefaults()
	cfg.LLM.Model = ""
	err := cfg.Validate()
	if err == nil {
		t.Errorf("expected validation to fail for missing model, got nil error")
	}
	if err.Error() != "llm.model is required" {
		t.Errorf("expected 'llm.model is required' error, got: %v", err)
	}
}

// TestValidateMissingProvider tests that validation fails without a provider
func TestValidateMissingProvider(t *testing.T) {
	cfg := LoadDefaults()
	cfg.LLM.Provider = ""
	err := cfg.Validate()
	if err == nil {
		t.Errorf("expected validation to fail for missing provider, got nil error")
	}
	if err.Error() != "llm.provider is required" {
		t.Errorf("expected 'llm.provider is required' error, got: %v", err)
	}
}

// TestValidateInvalidTemperature tests that validation fails for out-of-range temperature
func TestValidateInvalidTemperature(t *testing.T) {
	tests := []struct {
		name        string
		temperature float32
		shouldFail  bool
	}{
		{"negative temperature", -0.1, true},
		{"temperature zero", 0, false},
		{"temperature 1", 1, false},
		{"temperature 2", 2, false},
		{"temperature exceeds max", 2.1, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := LoadDefaults()
			cfg.LLM.Temperature = test.temperature
			err := cfg.Validate()

			if test.shouldFail && err == nil {
				t.Errorf("expected validation to fail for temperature %f", test.temperature)
			}
			if !test.shouldFail && err != nil {
				t.Errorf("expected validation to pass for temperature %f, got error: %v", test.temperature, err)
			}
		})
	}
}

// TestValidateInvalidPort tests that validation fails for invalid port numbers
func TestValidateInvalidPort(t *testing.T) {
	tests := []struct {
		name       string
		port       int
		shouldFail bool
	}{
		{"valid port low", 1, false},
		{"valid port mid", 8080, false},
		{"valid port high", 65535, false},
		{"port zero", 0, true},
		{"port negative", -1, true},
		{"port exceeds max", 65536, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := LoadDefaults()
			cfg.LLM.Local.Port = test.port
			err := cfg.Validate()

			if test.shouldFail && err == nil {
				t.Errorf("expected validation to fail for port %d", test.port)
			}
			if !test.shouldFail && err != nil {
				t.Errorf("expected validation to pass for port %d, got error: %v", test.port, err)
			}
		})
	}
}

// TestValidateInvalidLoggingLevel tests that validation fails for invalid logging levels
func TestValidateInvalidLoggingLevel(t *testing.T) {
	tests := []struct {
		name       string
		level      string
		shouldFail bool
	}{
		{"valid debug", "debug", false},
		{"valid info", "info", false},
		{"valid warn", "warn", false},
		{"valid error", "error", false},
		{"invalid level", "invalid", true},
		{"empty level", "", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := LoadDefaults()
			cfg.Logging.Level = test.level
			err := cfg.Validate()

			if test.shouldFail && err == nil {
				t.Errorf("expected validation to fail for level '%s'", test.level)
			}
			if !test.shouldFail && err != nil {
				t.Errorf("expected validation to pass for level '%s', got error: %v", test.level, err)
			}
		})
	}
}

// TestValidateMaxTokens tests that validation rejects non-positive MaxTokens
func TestValidateMaxTokens(t *testing.T) {
	tests := []struct {
		name       string
		maxTokens  int
		shouldFail bool
	}{
		{"positive tokens", 1000, false},
		{"zero tokens", 0, true},
		{"negative tokens", -1, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := LoadDefaults()
			cfg.LLM.MaxTokens = test.maxTokens
			err := cfg.Validate()

			if test.shouldFail && err == nil {
				t.Errorf("expected validation to fail for max_tokens %d", test.maxTokens)
			}
			if !test.shouldFail && err != nil {
				t.Errorf("expected validation to pass for max_tokens %d, got error: %v", test.maxTokens, err)
			}
		})
	}
}

// TestValidateTimeoutSeconds tests that validation rejects non-positive timeouts
func TestValidateTimeoutSeconds(t *testing.T) {
	tests := []struct {
		name       string
		timeout    int
		shouldFail bool
	}{
		{"positive timeout", 60, false},
		{"zero timeout", 0, true},
		{"negative timeout", -10, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := LoadDefaults()
			cfg.LLM.RequestTimeout = test.timeout
			err := cfg.Validate()

			if test.shouldFail && err == nil {
				t.Errorf("expected validation to fail for timeout %d", test.timeout)
			}
			if !test.shouldFail && err != nil {
				t.Errorf("expected validation to pass for timeout %d, got error: %v", test.timeout, err)
			}
		})
	}
}

// TestEnvironmentVariableOverrides tests that environment variables properly override config
func TestEnvironmentVariableOverrides(t *testing.T) {
	// Save original env vars
	origModel := os.Getenv("GOSHI_MODEL")
	origProvider := os.Getenv("GOSHI_LLM_PROVIDER")
	origURL := os.Getenv("GOSHI_OLLAMA_URL")
	origPort := os.Getenv("GOSHI_OLLAMA_PORT")

	defer func() {
		// Restore original env vars
		if origModel != "" {
			os.Setenv("GOSHI_MODEL", origModel)
		} else {
			os.Unsetenv("GOSHI_MODEL")
		}
		if origProvider != "" {
			os.Setenv("GOSHI_LLM_PROVIDER", origProvider)
		} else {
			os.Unsetenv("GOSHI_LLM_PROVIDER")
		}
		if origURL != "" {
			os.Setenv("GOSHI_OLLAMA_URL", origURL)
		} else {
			os.Unsetenv("GOSHI_OLLAMA_URL")
		}
		if origPort != "" {
			os.Setenv("GOSHI_OLLAMA_PORT", origPort)
		} else {
			os.Unsetenv("GOSHI_OLLAMA_PORT")
		}
		// Clear cache to prevent test interference
		cachedConfig = nil
	}()

	// Test GOSHI_MODEL override
	os.Setenv("GOSHI_MODEL", "gpt-4")
	os.Unsetenv("GOSHI_LLM_PROVIDER")
	os.Unsetenv("GOSHI_OLLAMA_URL")
	os.Unsetenv("GOSHI_OLLAMA_PORT")
	cachedConfig = nil

	cfg := Load()
	if cfg.LLM.Model != "gpt-4" {
		t.Errorf("expected GOSHI_MODEL to override model, got %s", cfg.LLM.Model)
	}

	// Test GOSHI_LLM_PROVIDER override
	os.Unsetenv("GOSHI_MODEL")
	os.Setenv("GOSHI_LLM_PROVIDER", "openai")
	os.Unsetenv("GOSHI_OLLAMA_URL")
	os.Unsetenv("GOSHI_OLLAMA_PORT")
	cachedConfig = nil

	cfg = Load()
	if cfg.LLM.Provider != "openai" {
		t.Errorf("expected GOSHI_LLM_PROVIDER to override provider, got %s", cfg.LLM.Provider)
	}

	// Test GOSHI_OLLAMA_URL override
	os.Unsetenv("GOSHI_MODEL")
	os.Unsetenv("GOSHI_LLM_PROVIDER")
	os.Setenv("GOSHI_OLLAMA_URL", "http://192.168.1.100")
	os.Unsetenv("GOSHI_OLLAMA_PORT")
	cachedConfig = nil

	cfg = Load()
	if cfg.LLM.Local.URL != "http://192.168.1.100" {
		t.Errorf("expected GOSHI_OLLAMA_URL to override URL, got %s", cfg.LLM.Local.URL)
	}

	// Test GOSHI_OLLAMA_PORT override
	os.Unsetenv("GOSHI_MODEL")
	os.Unsetenv("GOSHI_LLM_PROVIDER")
	os.Unsetenv("GOSHI_OLLAMA_URL")
	os.Setenv("GOSHI_OLLAMA_PORT", "9999")
	cachedConfig = nil

	cfg = Load()
	if cfg.LLM.Local.Port != 9999 {
		t.Errorf("expected GOSHI_OLLAMA_PORT to override port, got %d", cfg.LLM.Local.Port)
	}
}

// TestLoadCaching tests that Load() properly caches configuration
func TestLoadCaching(t *testing.T) {
	// Clear cache
	cachedConfig = nil

	cfg1 := Load()
	cfg2 := Load()

	// Both should reference the same underlying pointer
	if &cfg1 == &cfg2 {
		t.Errorf("expected separate struct values (copying), but got same pointer")
	}
}
