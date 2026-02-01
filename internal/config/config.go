package config

import "os"

type Config struct {
	Model  string
	LLMProvider string
  DryRun bool
  Yes bool
  JSON bool
}


func Load() Config {
	cfg := Config{
		Model:  os.Getenv("GROKGO_MODEL"),
		LLMProvider: os.Getenv("GROKGO_LLM_PROVIDER"),
		DryRun: true,
		Yes:    false,
    JSON: false,
	}

	if cfg.Model == "" {
		cfg.Model = "ollama"
	}

	if cfg.LLMProvider == "" {
		cfg.LLMProvider = "auto"
	}

	return cfg
}
