package llm

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cshaiku/goshi/internal/config"
)

func SelectProvider(cfg config.Config) string {
	// If provider is explicitly set to ollama, use it
	if cfg.LLM.Provider == "ollama" {
		return "ollama"
	}

	// If provider is explicitly set to openai, return it
	if cfg.LLM.Provider == "openai" {
		return "openai"
	}

	// Auto-detect: try ollama first
	if detectOllama(cfg) {
		return "ollama"
	}

	// Default to ollama even if not detected
	return "ollama"
}

func detectOllama(cfg config.Config) bool {
	client := http.Client{
		Timeout: 400 * time.Millisecond,
	}

	// Use configured URL and port
	url := cfg.LLM.Local.URL + ":" + fmt.Sprintf("%d", cfg.LLM.Local.Port) + "/api/tags"

	resp, err := client.Get(url)
	if err != nil {
		// Fallback to localhost:11434
		resp, err = client.Get("http://127.0.0.1:11434/api/tags")
		if err != nil {
			return false
		}
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}
