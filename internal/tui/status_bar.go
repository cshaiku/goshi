package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// StatusBar renders the two-line status bar at the bottom
type StatusBar struct {
	telemetry       *Telemetry
	lawsCount       int
	constraintCount int
	guardrailsOn    bool
}

// NewStatusBar creates a new status bar
func NewStatusBar(telemetry *Telemetry) *StatusBar {
	return &StatusBar{
		telemetry:    telemetry,
		guardrailsOn: true, // Default: guardrails active
	}
}

// UpdateMetrics updates the law and constraint counts
func (s *StatusBar) UpdateMetrics(laws, constraints int) {
	s.lawsCount = laws
	s.constraintCount = constraints
}

// SetGuardrails sets the guardrail status
func (s *StatusBar) SetGuardrails(enabled bool) {
	s.guardrailsOn = enabled
}

// Render returns the two-line status bar
func (s *StatusBar) Render(width int) string {
	line1 := s.renderLine1()
	line2 := s.renderLine2()

	// Style the status bar
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(width).
		Padding(0, 1)

	return style.Render(line1) + "\n" + style.Render(line2)
}

// renderLine1 renders the first status line
// Format: goshi | Laws: 312 | C: 9 | ACTIVE | 8k/16k | temp: 0.2 | mem: 14/128
func (s *StatusBar) renderLine1() string {
	tokensUsedK := s.telemetry.TokensUsed / 1024
	tokensLimitK := s.telemetry.TokensLimit / 1024

	return fmt.Sprintf(
		"goshi │ Laws: %d │ C: %d │ %s │ %dk/%dk │ temp: %.1f │ mem: %d/%d",
		s.lawsCount,
		s.constraintCount,
		s.telemetry.Status,
		tokensUsedK,
		tokensLimitK,
		s.telemetry.Temperature,
		s.telemetry.MemoryEntries,
		s.telemetry.MemoryMax,
	)
}

// renderLine2 renders the second status line
// Format: lat: 423ms | cost: $0.0031 | guard: ON | llm: ollama | model: qwen2.5-coder-7b
func (s *StatusBar) renderLine2() string {
	guardStatus := "ON"
	if !s.guardrailsOn {
		guardStatus = "OFF"
	}

	return fmt.Sprintf(
		"lat: %dms │ cost: $%.4f │ guard: %s │ llm: %s │ model: %s",
		s.telemetry.LatencyMS(),
		s.telemetry.SessionCost,
		guardStatus,
		s.telemetry.Backend,
		s.telemetry.ModelName,
	)
}
