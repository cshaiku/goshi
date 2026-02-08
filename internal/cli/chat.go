package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cshaiku/goshi/internal/config"
	"github.com/cshaiku/goshi/internal/llm"
	"github.com/cshaiku/goshi/internal/llm/ollama"
)

// runChat starts an interactive REPL-style chat session.
// It blocks on stdin until the user exits.
func runChat(systemPrompt string) {
	cfg := config.Load()
	ctx := context.Background()

	// Select LLM client (local-first)
	var client llm.Client
	switch cfg.LLMProvider {
	case "ollama", "", "auto":
		client = ollama.New(cfg.Model)
	default:
		fmt.Fprintf(os.Stderr, "unsupported LLM provider: %s\n", cfg.LLMProvider)
		return
	}

	reader := bufio.NewReader(os.Stdin)

	// Conversation state
	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
	}

	fmt.Println(systemPrompt)
	fmt.Println("Type /quit to exit.")
	fmt.Println("-----------------------------------------------------")

	for {
		fmt.Print("You: ")

		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("\nExiting.")
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "/quit" {
			return
		}

		messages = append(messages, llm.Message{
			Role:    "user",
			Content: line,
		})

		stream, err := client.Stream(ctx, messages)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}

		fmt.Print("Goshi: ")
		var reply strings.Builder

		for {
			chunk, err := stream.Recv()
			if err != nil {
				break
			}
			fmt.Print(chunk)
			reply.WriteString(chunk)
		}
		fmt.Println()

		_ = stream.Close()

		messages = append(messages, llm.Message{
			Role:    "assistant",
			Content: reply.String(),
		})
	}
}
