// cmd/grokgo/main.go
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ZaguanLabs/xai-sdk-go/xai"
	"github.com/ZaguanLabs/xai-sdk-go/xai/chat"
)

func main() {
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		log.Fatal("XAI_API_KEY environment variable is not set. Get it from https://console.grok.x.ai and export it.")
	}

	model := os.Getenv("GROKGO_MODEL")
	if model == "" {
		model = "grok-beta" // Try this first; change to "grok-3-mini" or console-confirmed name if fails
	}
	log.Printf("Using model: %s (override with export GROKGO_MODEL=...)", model)

	// Use library defaults (positive timeouts)
	config := xai.DefaultConfig()
	config.APIKey = apiKey

	client, err := xai.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Conversation history
	var messages []*chat.Message
	messages = append(messages, chat.System(chat.Text("You are Grok, a helpful and maximally truthful AI built by xAI.")))

	fmt.Println("GrokGo Basic Chat Test (streaming)")
	fmt.Println("Type your message and press Enter. Type /quit to exit.")
	fmt.Println("-----------------------------------------------------")

	scanner := bufio.NewScanner(os.Stdin)
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

		messages = append(messages, chat.User(chat.Text(userInput)))

		req := chat.NewRequest(
			model,
			chat.WithTemperature(0.7),
			chat.WithMaxTokens(2048),
			chat.WithMessages(messages...),
		)

		stream, err := req.Stream(context.Background(), client.Chat())
		if err != nil {
			log.Printf("Stream failed: %v\nHint: Check model name '%s' in https://console.grok.x.ai or try 'grok-3-mini'.", err, model)
			continue
		}
		defer stream.Close()

		fmt.Print("Grok: ")
		var assistantContent strings.Builder
		for stream.Next() {
			chunk := stream.Current()
			if content := chunk.Content(); content != "" {
				fmt.Print(content)
				assistantContent.WriteString(content)
			}
		}
		fmt.Println()

		if assistantContent.Len() > 0 {
			messages = append(messages, chat.Assistant(chat.Text(assistantContent.String())))
		}

		if err := stream.Err(); err != nil {
			log.Printf("Stream error: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Input error: %v", err)
	}
}
