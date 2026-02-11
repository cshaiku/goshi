package app

import (
	"errors"

	"github.com/cshaiku/goshi/internal/llm"
)

var errSystemPromptMissing = errors.New("system prompt not provided")

type App struct {
	LLM *llm.Client
}

func New(systemPrompt *llm.SystemPrompt, backend llm.Backend) (*App, error) {
	if systemPrompt == nil {
		return nil, errSystemPromptMissing
	}

	client := llm.NewClient(systemPrompt, backend)
	return &App{LLM: client}, nil
}
