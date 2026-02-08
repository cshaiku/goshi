package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cshaiku/goshi/internal/config"
	"github.com/cshaiku/goshi/internal/detect"
	"github.com/cshaiku/goshi/internal/llm"
	"github.com/cshaiku/goshi/internal/llm/ollama"
	"github.com/cshaiku/goshi/internal/selfmodel"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)

// printStatus prints the self-model status without invoking the LLM.
func printStatus(systemPrompt string) {
	metrics := selfmodel.ComputeLawMetrics(systemPrompt)

	enforcementLabel := "ENFORCEMENT STAGED"
	enforcementColor := colorYellow
	if metrics.EnforcementActive {
		enforcementLabel = "ENFORCEMENT ACTIVE"
		enforcementColor = colorGreen
	}

	fmt.Printf(
		"Self-Model Law Index: %d lines · %d constraints · %s%s%s\n",
		metrics.RuleLines,
		metrics.ConstraintCount,
		enforcementColor,
		enforcementLabel,
		colorReset,
	)

	if laws := selfmodel.ExtractPrimaryLaws(systemPrompt); len(laws) > 0 {
		fmt.Printf("Primary Laws: %s\n", strings.Join(laws, " · "))
	}

	if greeting := selfmodel.ExtractHumanGreeting(systemPrompt); greeting != "" {
		fmt.Println()
		fmt.Println(greeting)
	}

	fmt.Println("-----------------------------------------------------")
}

// refuseFSRead emits a static refusal for filesystem reads.
func refuseFSRead() {
	fmt.Println("I do not have access to your local filesystem.")
	fmt.Println("To read files, you must provide command output explicitly.")
	fmt.Println("Example: run `ls` or `cat file` and paste the result.")
	fmt.Println("-----------------------------------------------------")
}

// runChat starts an interactive REPL-style chat session.
// It blocks on stdin until the user exits.
func runChat(systemPrompt string) {
	cfg := config.Load()
	ctx := context.Background()

	// Select LLM backend (local-first)
	var backend llm.Backend
	switch cfg.LLMProvider {
	case "ollama", "", "auto":
		backend = ollama.New(cfg.Model)
	default:
		fmt.Fprintf(os.Stderr, "unsupported LLM provider: %s\n", cfg.LLMProvider)
		return
	}

	sp, err := llm.NewSystemPrompt(systemPrompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid system prompt: %v\n", err)
		return
	}

	client := llm.NewClient(sp, backend)

	// Initial status on startup
	printStatus(systemPrompt)

	reader := bufio.NewReader(os.Stdin)
	messages := []llm.Message{}

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

		// Local commands (no LLM)
		switch line {
		case "/quit":
			return
		case "/status":
			printStatus(systemPrompt)
			continue
		}

		// --- STEP 2: CAPABILITY GATING (fs_read) ---
		blocked := false
		caps := detect.DetectCapabilities(line, detect.FSReadRules)
		for _, cap := range caps {
			if cap == detect.CapabilityFSRead {
				refuseFSRead()
				blocked = true
				break
			}
		}
		if blocked {
			continue
		}

		// Normal LLM path
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
