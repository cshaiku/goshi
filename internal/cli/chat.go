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
	"github.com/cshaiku/goshi/internal/selfmodel"
	"github.com/cshaiku/goshi/internal/session"
	"github.com/cshaiku/goshi/internal/tui"
)

func printStatus(systemPrompt string, perms *session.Permissions) {
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

// runTUIMode starts the TUI (Text User Interface) mode
func runTUIMode(systemPrompt string) {
	cfg := config.Load()
	ctx := context.Background()

	// Initialize LLM backend
	factory := NewBackendFactory(cfg.LLMProvider, cfg.Model)
	backend, err := factory.Create()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize LLM backend: %v\n", err)
		fmt.Fprintf(os.Stderr, "supported providers: %s\n", strings.Join(SupportedProviders(), ", "))
		return
	}

	// Create chat session
	sess, err := session.NewChatSession(ctx, systemPrompt, backend)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize chat session: %v\n", err)
		return
	}

	// Launch TUI
	if err := tui.Run(systemPrompt, sess); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
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
	sess, err := session.NewChatSession(ctx, systemPrompt, backend)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize chat session: %v\n", err)
		return
	}

	printStatus(systemPrompt, sess.Permissions)
	reader := bufio.NewReader(os.Stdin)
	permHandler := NewPermissionHandler(sess.WorkingDir, DefaultDisplayConfig())

	for {
		fmt.Print("You: ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// PHASE 1: Listen - Record user input
		sess.AddUserMessage(line)

		// PHASE 2: Detect intent - Check for implicit capability requests
		// This is a transition mechanism; eventually LLM should handle all intent
		detected := detect.DetectCapabilities(line, detect.FSReadRules)
		detected = append(detected, detect.DetectCapabilities(line, detect.FSWriteRules)...)

		// Handle permissions using extracted handler (Single Responsibility)
		if !permHandler.HandleDetected(detected, sess, systemPrompt) {
			continue
		}

		// PHASE 3: Plan - Get LLM response with streaming
		collector := llm.NewResponseCollector(llm.NewStructuredParser())
		stream, err := sess.Client.Backend().Stream(ctx, sess.Client.System().Raw(), sess.ConvertMessagesToLegacy())
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

		// Parse response
		parseResult, parseErr := collector.Parse()
		if parseErr != nil || parseResult == nil {
			fmt.Println("-----------------------------------------------------")
			continue
		}

		// Store text response in session
		if textContent := parseResult.Response.Text; textContent != "" {
			sess.AddAssistantTextMessage(textContent)
		}

		// TODO Phase 4: Full tool execution integration
		// For now, skip actions - Phase 3 focuses on TUI mode detection

		fmt.Println("-----------------------------------------------------")
	}
}
