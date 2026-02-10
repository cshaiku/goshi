// LoadDefaults returns a Config with safe defaults
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
