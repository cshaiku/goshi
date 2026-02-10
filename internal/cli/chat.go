package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cshaiku/goshi/internal/app"
	"github.com/cshaiku/goshi/internal/config"
	"github.com/cshaiku/goshi/internal/detect"
	"github.com/cshaiku/goshi/internal/llm"
	"github.com/cshaiku/goshi/internal/llm/ollama"
	"github.com/cshaiku/goshi/internal/llm/openai"
	"github.com/cshaiku/goshi/internal/selfmodel"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)

func printStatus(systemPrompt string, perms *Permissions) {
	metrics := selfmodel.ComputeLawMetrics(systemPrompt)
	label := "ENFORCEMENT STAGED"
	color := colorYellow
	if perms.FSRead && perms.FSWrite {
		label = "ENFORCEMENT ACTIVE (FS_READ + FS_WRITE)"
		color = colorGreen
	} else if perms.FSRead {
		label = "ENFORCEMENT ACTIVE (FS_READ)"
		color = colorGreen
	} else if perms.FSWrite {
		label = "ENFORCEMENT ACTIVE (FS_WRITE)"
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

func refuseFSWrite() {
	fmt.Println("Filesystem write access denied.\nPermission was not granted for this session.")
	fmt.Println("-----------------------------------------------------")
}

func runChat(systemPrompt string) {
	cfg := config.Load()
	ctx := context.Background()

	// Initialize LLM backend
	var backend llm.Backend
	var err error
	
	if cfg.LLMProvider == "ollama" || cfg.LLMProvider == "" || cfg.LLMProvider == "auto" {
		backend = ollama.New(cfg.Model)
	} else if cfg.LLMProvider == "openai" {
		backend, err = openai.New(cfg.Model)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialize OpenAI backend: %v\n", err)
			return
		}
	} else {
		fmt.Fprintf(os.Stderr, "unsupported LLM provider: %s\n", cfg.LLMProvider)
		fmt.Fprintf(os.Stderr, "supported providers: ollama, openai\n")
		return
	}

	// Create session encapsulating all chat context
	session, err := NewChatSession(ctx, systemPrompt, backend)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize chat session: %v\n", err)
		return
	}

	printStatus(systemPrompt, session.Permissions)
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("You: ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// PHASE 1: Listen - Record user input
		session.AddUserMessage(line)

		// PHASE 2: Detect intent - Check for implicit capability requests via regex
		// This is a transition mechanism; eventually LLM should handle all intent
		detected := detect.DetectCapabilities(line, detect.FSReadRules)
		detected = append(detected, detect.DetectCapabilities(line, detect.FSWriteRules)...)

		permissionDenied := false
		for _, cap := range detected {
			if cap == detect.CapabilityFSRead && !session.HasPermission("FS_READ") {
				if !RequestFSReadPermission(session.WorkingDir) {
					refuseFSRead()
					permissionDenied = true
					session.DenyPermission("FS_READ")
					break
				}
				session.GrantPermission("FS_READ")
				printStatus(systemPrompt, session.Permissions)
			}
			if cap == detect.CapabilityFSWrite && !session.HasPermission("FS_WRITE") {
				if !RequestFSWritePermission(session.WorkingDir) {
					refuseFSWrite()
					permissionDenied = true
					session.DenyPermission("FS_WRITE")
					break
				}
				session.GrantPermission("FS_WRITE")
				printStatus(systemPrompt, session.Permissions)
			}
		}

		if permissionDenied {
			continue
		}

		// PHASE 3: Plan - Get LLM response with streaming
		collector := llm.NewResponseCollector(llm.NewStructuredParser())
		stream, err := session.Client.Backend().Stream(ctx, session.Client.System().Raw(), session.ConvertMessagesToLegacy())
		if err != nil {
			fmt.Fprintf(os.Stderr, "LLM error: %v\n", err)
			continue
		}

		fmt.Print("Goshi: ")
		for {
			chunk, err := stream.Recv()
			if err != nil {
				break
			}
			fmt.Print(chunk)
			collector.AddChunk(chunk)
		}
		fmt.Println()
		stream.Close()

		// PHASE 4: Parse - Extract structured response and validate tool calls
		response := collector.GetFullResponse()
		session.AddAssistantTextMessage(response) // Log the full response first

		parseResult, err := collector.Parse()
		if err != nil || parseResult == nil {
			// No structured tool call detected - just text response
			fmt.Println("-----------------------------------------------------")
			continue
		}

		// Check if this is a tool action response
		if parseResult.Response.Type != llm.ResponseTypeAction || parseResult.Response.Action == nil {
			// Not a tool action - just text
			fmt.Println("-----------------------------------------------------")
			continue
		}

		// PHASE 5: Act - Execute the tool with validation
		toolName := parseResult.Response.Action.Tool
		toolArgs := parseResult.Response.Action.Args

		// Validate the tool call against schema and permissions
		result := session.ToolRouter.Handle(app.ToolCall{
			Name: toolName,
			Args: toolArgs,
		})

		// Log tool execution and result
		session.AddAssistantActionMessage(toolName, toolArgs)
		session.AddToolResultMessage(toolName, result)

		fmt.Println("Goshi (Action Result):")
		printJSON(result)

		// PHASE 6: Report - Optional follow-up from LLM based on action result
		// For now, we'll skip this unless specifically implemented
		fmt.Println("-----------------------------------------------------")
	}
}
