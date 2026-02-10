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
	"github.com/cshaiku/goshi/internal/selfmodel"
)

func printStatus(systemPrompt string, perms *Permissions) {
	display := DefaultDisplayConfig()
	metrics := selfmodel.ComputeLawMetrics(systemPrompt)
	label := "ENFORCEMENT STAGED"
	color := ColorYellow
	if perms.FSRead && perms.FSWrite {
		label = "ENFORCEMENT ACTIVE (FS_READ + FS_WRITE)"
		color = ColorGreen
	} else if perms.FSRead {
		label = "ENFORCEMENT ACTIVE (FS_READ)"
		color = ColorGreen
	} else if perms.FSWrite {
		label = "ENFORCEMENT ACTIVE (FS_WRITE)"
		color = ColorGreen
	}
	fmt.Printf("Self-Model Law Index: %d lines · %d constraints · %s\n",
		metrics.RuleLines, metrics.ConstraintCount, display.Colorize(label, color))
	if laws := selfmodel.ExtractPrimaryLaws(systemPrompt); len(laws) > 0 {
		fmt.Printf("Primary Laws: %s\n", strings.Join(laws, " · "))
	}
	if greeting := selfmodel.ExtractHumanGreeting(systemPrompt); greeting != "" {
		fmt.Println("\n" + greeting)
	}
	fmt.Println("-----------------------------------------------------")
}

func runChat(systemPrompt string) {
	cfg := config.Load()
	ctx := context.Background()

	// Initialize LLM backend using factory (Dependency Inversion Principle)
	factory := NewBackendFactory(cfg.LLMProvider, cfg.Model)
	backend, err := factory.Create()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize LLM backend: %v\n", err)
		fmt.Fprintf(os.Stderr, "supported providers: %s\n", strings.Join(SupportedProviders(), ", "))
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
	permHandler := NewPermissionHandler(session.WorkingDir, DefaultDisplayConfig())

	for {
		fmt.Print("You: ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// PHASE 1: Listen - Record user input
		session.AddUserMessage(line)

		// PHASE 2: Detect intent - Check for implicit capability requests
		// This is a transition mechanism; eventually LLM should handle all intent
		detected := detect.DetectCapabilities(line, detect.FSReadRules)
		detected = append(detected, detect.DetectCapabilities(line, detect.FSWriteRules)...)

		// Handle permissions using extracted handler (Single Responsibility)
		if !permHandler.HandleDetected(detected, session, systemPrompt) {
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
