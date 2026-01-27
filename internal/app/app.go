package app

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

  "grokgo/internal/llm"
  "grokgo/internal/llm/xai"
  "grokgo/internal/config"
)

func Run(cfg config.Config) {
  if cfg.APIKey == "" {
  	log.Fatal("XAI_API_KEY environment variable is not set. Get it from https://console.grok.x.ai and export it.")
  }

  log.Printf("Using model: %s (override with export GROKGO_MODEL=...)", cfg.Model)

  client, err := xai.New(cfg.APIKey, cfg.Model)
  if err != nil {
  	log.Fatalf("Failed to create client: %v", err)
  }

  var messages []llm.Message
  messages = append(messages, llm.Message{
  	Role:    "system",
  	Content: "You are Grok, a helpful and maximally truthful AI built by xAI.",
  })

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

    for stream.Next() {
    	if content := stream.Content(); content != "" {
    		fmt.Print(content)
    		assistantContent.WriteString(content)
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

    if err := stream.Err(); err != nil {
    	log.Printf("Stream error: %v", err)
    }

  	if err := scanner.Err(); err != nil {
		  log.Printf("Input error: %v", err)
	  }
  }

}
