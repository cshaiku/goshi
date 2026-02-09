package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

func printStatus(systemPrompt string, perms Permissions) {
	metrics := selfmodel.ComputeLawMetrics(systemPrompt)
	label := "ENFORCEMENT STAGED"
	color := colorYellow
	if perms.FSRead {
		label = "ENFORCEMENT ACTIVE (FS_READ)"
		color = colorGreen
	}
	fmt.Printf("Self-Model Law Index: %d lines · %d constraints · %s%s%s\n",
		metrics.RuleLines, metrics.ConstraintCount, color, label, colorReset)
	if laws := selfmodel.ExtractPrimaryLaws(systemPrompt); len(laws) > 0 {
		fmt.Printf("Primary Laws: %s\n", strings.Join(laws, " · "))
	}
	if greeting := selfmodel.ExtractHumanGreeting(systemPrompt); greeting != "" {
		fmt.Println("\n" + greeting)
	}
	fmt.Println("-----------------------------------------------------")
}

func refuseFSRead() {
	fmt.Println("Filesystem access denied.\nPermission was not granted for this session.")
	fmt.Println("-----------------------------------------------------")
}

func runChat(systemPrompt string) {
	cfg := config.Load()
	ctx := context.Background()
	var backend llm.Backend
	if cfg.LLMProvider == "ollama" || cfg.LLMProvider == "" || cfg.LLMProvider == "auto" {
		backend = ollama.New(cfg.Model)
	} else {
		fmt.Fprintf(os.Stderr, "unsupported LLM provider\n")
		return
	}

	sp, _ := llm.NewSystemPrompt(systemPrompt)
	client := llm.NewClient(sp, backend)
	caps := app.NewCapabilities()
	perms := Permissions{}
	cwd, _ := os.Getwd()
	cwd, _ = filepath.EvalSymlinks(cwd)
	actionSvc, _ := app.NewActionService(cwd)
	router := app.NewToolRouter(actionSvc.Dispatcher(), caps)

	printStatus(systemPrompt, perms)
	reader := bufio.NewReader(os.Stdin)
	messages := []llm.Message{}

	for {
		fmt.Print("You: ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" { continue }

		detected := detect.DetectCapabilities(line, detect.FSReadRules)
		handled := false

		if len(detected) > 0 {
			for _, cap := range detected {
				if cap == detect.CapabilityFSRead && !perms.FSRead {
					if !RequestFSReadPermission(cwd) {
						refuseFSRead()
						handled = true
						break
					}
					perms.FSRead = true
					caps.Grant(app.CapFSRead)
					printStatus(systemPrompt, perms)
				}
			}

			if perms.FSRead && !handled {
				// Robust regex with non-capturing groups for filler words
				re := regexp.MustCompile(`(?i)(read|list|dir)(?:\s+(?:the|this|in|at|files?|folders?|dirs?|paths?))*\s*([^\s]+)?`)
				matches := re.FindStringSubmatch(line)

				if len(matches) > 1 {
					verb := strings.ToLower(matches[3])
					toolName := "fs.list"
					toolPath := "."
					if verb == "read" { toolName = "fs.read" }
					if len(matches) > 2 && matches[4] != "" { toolPath = matches[4] }

					result := router.Handle(app.ToolCall{Name: toolName, Args: map[string]any{"path": toolPath}})
					fmt.Println("Goshi (Direct Action):")
					printJSON(result) // Uses printJSON from fs.go
					fmt.Println("-----------------------------------------------------")
					handled = true
				}
			}
		}

		if handled { continue }

		messages = append(messages, llm.Message{Role: "user", Content: line})
		stream, err := client.Stream(ctx, messages)
		if err != nil { continue }

		fmt.Print("Goshi: ")
		var reply strings.Builder
		for {
			chunk, err := stream.Recv()
			if err != nil { break }
			fmt.Print(chunk)
			reply.WriteString(chunk)
		}
		fmt.Println()
		stream.Close()

		toolMsg, toolHandled := app.TryHandleToolCall(router, reply.String())
		if toolHandled {
			messages = append(messages, *toolMsg)
			if stream, err = client.Stream(ctx, messages); err == nil {
				fmt.Print("Goshi (Final): ")
				for {
					chunk, err := stream.Recv()
					if err != nil { break }
					fmt.Print(chunk)
				}
				fmt.Println()
				stream.Close()
			}
		} else {
			messages = append(messages, llm.Message{Role: "assistant", Content: reply.String()})
		}
		fmt.Println("-----------------------------------------------------")
	}
}
