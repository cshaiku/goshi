package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cshaiku/goshi/internal/app"
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
func printStatus(systemPrompt string, perms Permissions) {
	metrics := selfmodel.ComputeLawMetrics(systemPrompt)

	enforcementLabel := "ENFORCEMENT STAGED"
	enforcementColor := colorYellow
	if perms.FSRead {
		enforcementLabel = "ENFORCEMENT ACTIVE (FS_READ)"
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

func refuseFSRead() {
	fmt.Println("Filesystem access denied.")
	fmt.Println("Permission was not granted for this session.")
	fmt.Println("-----------------------------------------------------")
}

// runChat starts an interactive REPL-style chat session.
func runChat(systemPrompt string) {
	cfg := config.Load()
	ctx := context.Background()

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

	// --- STEP 2: SESSION CAPABILITIES + ACTION WIRING ---

	caps := app.NewCapabilities()

	cwd, _ := os.Getwd()
	cwd, _ = filepath.EvalSymlinks(cwd)

	actionSvc, err := app.NewActionService(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize actions: %v\n", err)
		return
	}

	router := app.NewToolRouter(actionSvc.Dispatcher(), caps)

	// ---------------------------------------------------

	perms := Permissions{}

	printStatus(systemPrompt, perms)

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

		switch line {
		case "/quit":
			return
		case "/status":
			printStatus(systemPrompt, perms)
			continue
		}

		// Capability detection + permission UI
		blocked := false
		detected := detect.DetectCapabilities(line, detect.FSReadRules)
		for _, cap := range detected {
			if cap == detect.CapabilityFSRead && !perms.FSRead {
				allowed := RequestFSReadPermission(cwd)
				if !allowed {
					refuseFSRead()
					blocked = true
					break
				}
				perms.FSRead = true
				caps.Grant(app.CapFSRead)
				printStatus(systemPrompt, perms)
			}
		}
		if blocked {
			continue
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

		// Tool handling (already existing logic)
		if msg, ok := app.TryHandleToolCall(router, reply.String()); ok {
			messages = append(messages, *msg)
			continue
		}

		messages = append(messages, llm.Message{
			Role:    "assistant",
			Content: reply.String(),
		})
	}
}
