package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cshaiku/goshi/internal/app"
	"github.com/cshaiku/goshi/internal/audit"
	"github.com/cshaiku/goshi/internal/config"
	"github.com/cshaiku/goshi/internal/llm"
)

// ChatSession encapsulates a single chat interaction session with all necessary context
// This manages message history, permissions, and conversation state
type ChatSession struct {
	SystemPrompt string
	WorkingDir   string
	Permissions  *Permissions
	Capabilities *app.Capabilities
	Messages     []llm.LLMMessage // Structured message history
	Client       *llm.ClientWithTools
	ToolRouter   *app.ToolRouter
	AuditLogger  *audit.Logger
	Context      context.Context
	Model        string // LLM model name
	Provider     string // LLM provider name
}

// NewChatSession initializes a new chat session with the given system prompt
func NewChatSession(ctx context.Context, systemPrompt string, backend llm.Backend) (*ChatSession, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	cwd, err = filepath.EvalSymlinks(cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to eval symlinks: %w", err)
	}

	// Initialize system prompt
	sp, err := llm.NewSystemPrompt(systemPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to create system prompt: %w", err)
	}

	// Initialize LLM client with tools support
	client := llm.NewClientWithTools(sp, backend)

	// Initialize capabilities and permissions
	caps := app.NewCapabilities()
	cfg := config.Load()
	repoRoot := cfg.Behavior.RepoRoot
	if repoRoot == "" {
		repoRoot = cwd
	}

	auditLogger, err := audit.NewLogger(audit.Config{
		Enabled:            cfg.Audit.Enabled,
		Dir:                cfg.Audit.Dir,
		RetentionDays:      cfg.Audit.RetentionDays,
		MaxSessions:        cfg.Audit.MaxSessions,
		Redact:             cfg.Audit.Redact,
		ToolArgumentsStyle: cfg.Audit.ToolArgumentsStyle,
	}, repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize audit logger: %w", err)
	}
	perms := &Permissions{
		AuditLog: []PermissionEntry{},
		Logger:   auditLogger,
	}

	// Initialize action service and tool router
	actionSvc, err := app.NewActionService(cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to create action service: %w", err)
	}

	router := app.NewToolRouter(actionSvc.Dispatcher(), caps)
	router.SetAuditLogger(auditLogger, cwd)
	if auditLogger != nil {
		auditLogger.LogSession("START", fmt.Sprintf("session started (provider=%s model=%s)", cfg.LLM.Provider, cfg.LLM.Model), cwd)
	}

	// Set up tool validation in the parser
	client.SetToolValidator(func(toolName string, args map[string]any) error {
		return router.ValidateToolCall(toolName, args)
	})

	return &ChatSession{
		SystemPrompt: systemPrompt,
		WorkingDir:   cwd,
		Permissions:  perms,
		Capabilities: caps,
		Messages:     []llm.LLMMessage{},
		Client:       client,
		ToolRouter:   router,
		AuditLogger:  auditLogger,
		Context:      ctx,
		Model:        cfg.LLM.Model,
		Provider:     cfg.LLM.Provider,
	}, nil
}

// AddUserMessage adds a user message to the conversation history
func (s *ChatSession) AddUserMessage(content string) {
	msg := llm.UserMessage{
		Content: content,
	}
	s.Messages = append(s.Messages, &msg)

	// Log user message
	if s.AuditLogger != nil {
		s.AuditLogger.LogMessage(content, s.WorkingDir)
	}
}

// AddAssistantTextMessage adds an assistant text message to the conversation history
func (s *ChatSession) AddAssistantTextMessage(content string) {
	msg := llm.AssistantTextMessage{
		Content: content,
	}
	s.Messages = append(s.Messages, &msg)

	// Log LLM text response
	if s.AuditLogger != nil {
		s.AuditLogger.LogResponse(content, false, s.WorkingDir)
	}
}

// AddAssistantActionMessage adds an assistant action message to the conversation history
func (s *ChatSession) AddAssistantActionMessage(toolName string, toolArgs map[string]any) {
	msg := llm.AssistantActionMessage{
		ToolName: toolName,
		ToolArgs: toolArgs,
		ToolID:   "auto",
	}
	s.Messages = append(s.Messages, &msg)

	// Log LLM tool call response
	if s.AuditLogger != nil {
		s.AuditLogger.LogResponse(toolName, true, s.WorkingDir)
	}
}

// AddToolResultMessage adds a tool result message to the conversation history
func (s *ChatSession) AddToolResultMessage(toolName string, result interface{}) {
	msg := llm.ToolResultMessage{
		ToolName: toolName,
		Result:   result,
	}
	s.Messages = append(s.Messages, &msg)
}

// GrantPermission grants a capability and records it in the audit log
func (s *ChatSession) GrantPermission(capability string) {
	s.Permissions.Grant(capability, s.WorkingDir)
	switch capability {
	case "FS_READ":
		s.Capabilities.Grant(app.CapFSRead)
	case "FS_WRITE":
		s.Capabilities.Grant(app.CapFSWrite)
	}
}

// DenyPermission denies a capability and records it in the audit log
func (s *ChatSession) DenyPermission(capability string) {
	s.Permissions.Deny(capability, s.WorkingDir)
}

// HasPermission checks if a capability is currently granted
func (s *ChatSession) HasPermission(capability string) bool {
	return s.Permissions.HasPermission(capability)
}

// GetAuditLog returns the full audit trail
func (s *ChatSession) GetAuditLog() string {
	return s.Permissions.GetAuditTrail()
}

// ConvertMessagesToLegacy converts structured LLMMessages back to legacy Message format
// This is temporary for backward compatibility during transition
func (s *ChatSession) ConvertMessagesToLegacy() []llm.Message {
	var legacyMessages []llm.Message

	for _, msg := range s.Messages {
		if userMsg, ok := msg.(*llm.UserMessage); ok {
			legacyMessages = append(legacyMessages, llm.Message{
				Role:    "user",
				Content: userMsg.Content,
			})
		} else if assistantMsg, ok := msg.(*llm.AssistantTextMessage); ok {
			legacyMessages = append(legacyMessages, llm.Message{
				Role:    "assistant",
				Content: assistantMsg.Content,
			})
		}
		// Note: We'll need to handle action messages differently in the chat loop
	}

	return legacyMessages
}
