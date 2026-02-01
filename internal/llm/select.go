package llm

import (
	"net/http"
	"time"

	"grokgo/internal/config"
)

func SelectProvider(cfg config.Config) string {
	return "ollama"
}

func detectOllama() bool {
	client := http.Client{
		Timeout: 400 * time.Millisecond,
	}

	resp, err := client.Get("http://127.0.0.1:11434/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}
