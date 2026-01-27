package config

import "os"

type Config struct {
	APIKey string
	Model  string
  DryRun bool
  Yes bool
}


func Load() Config {
	cfg := Config{
		APIKey: os.Getenv("XAI_API_KEY"),
		Model:  os.Getenv("GROKGO_MODEL"),
		DryRun: true,
		Yes:    false,
	}

	if cfg.Model == "" {
		cfg.Model = "grok-beta"
	}

	return cfg
}
