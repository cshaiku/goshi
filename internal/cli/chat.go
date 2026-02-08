package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/cshaiku/goshi/internal/config"
	"github.com/cshaiku/goshi/internal/llm"
	"github.com/cshaiku/goshi/internal/llm/ollama"
)

// runChat owns LLM client creation and streaming.
func runChat(ctx context.Context, cfg config.Config, systemPrompt string) {
	var client llm.Client

	switch llm.SelectProvider(cfg) {
	case "ollama":
		// ollama.New takes a model string and returns *ollama.Client
		client = ollama.New(cfg.Model)
	default:
		panic(errors.New("no supported LLM provider available"))
	}

	messages := []llm.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	stream, err := client.Stream(ctx, messages)
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	for {
		chunk, err := stream.Recv()
		if err != nil {
			break
		}
		fmt.Print(chunk)
	}
}
