package app

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"grokgo/internal/config"
	"grokgo/internal/llm"
	"grokgo/internal/llm/ollama"
)

func Run(cfg config.Config) {
	// Fail fast if Ollama is not running
	healthCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := ollama.CheckHealth(healthCtx); err != nil {
		log.Fatalf("Ollama health check failed: %v", err)
	}

	var client llm.Client
	client = ollama.New(cfg.Model)

	var messages []llm.Message
	messages = append(messages, llm.Message{
		Role:    "system",
		Content: "You are Grok, a helpful and maximally truthful AI built by xAI.",
	})

	fmt.Println("GrokGo Basic Chat Test (streaming)")
	fmt.Println("Type your message and press Enter. Type /quit to exit.")
	fmt.Println("-----------------------------------------------------")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for {
		fmt.Print("\nYou: ")
		if !scanner.Scan() {
			break
		}

		userInput := strings.TrimSpace(scanner.Text())
		if userInput == "" || userInput == "/quit" {
			fmt.Println("Goodbye!")
			break
		}

		messages = append(messages, llm.Message{
			Role:    "user",
			Content: userInput,
		})

		stream, err := client.Stream(context.Background(), messages)
		if err != nil {
			log.Printf("Stream failed: %v", err)
			continue
		}

		fmt.Print("Grok: ")
		var assistantContent strings.Builder

		for {
			chunk, err := stream.Recv()
			if err != nil {
				break
			}
			if chunk != "" {
				fmt.Print(chunk)
				assistantContent.WriteString(chunk)
			}
		}

		stream.Close()
		fmt.Println()

		if assistantContent.Len() > 0 {
			messages = append(messages, llm.Message{
				Role:    "assistant",
				Content: assistantContent.String(),
			})
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Input error: %v", err)
	}
}
