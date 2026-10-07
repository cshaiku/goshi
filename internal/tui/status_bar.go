package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// StatusBar renders the single-line status bar at the bottom
type StatusBar struct {
	telemetry *Telemetry
	mode      string
}

// NewStatusBar creates a new status bar
func NewStatusBar(telemetry *Telemetry) *StatusBar {
	return &StatusBar{
		telemetry: telemetry,
	}
}

// UpdateMetrics updates the law and constraint counts
func (s *StatusBar) UpdateMetrics(laws, constraints int) {
	// No-op: status bar no longer renders law/constraint counts.
}

// SetGuardrails sets the guardrail status
func (s *StatusBar) SetGuardrails(enabled bool) {
	// No-op: status bar no longer renders guardrails state.
}

// UpdateMode updates the active mode label.
func (s *StatusBar) UpdateMode(mode string) {
	s.mode = mode
}

// Render returns the single-line status bar
func (s *StatusBar) Render(width int) string {
	line := s.renderLine()

	// Style the status bar
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(width).
		Padding(0, 1)

	return style.Render(line)
}

// renderLine renders the status line
// Format: Chat | latency: 423ms | cost: $0.0031 | llm: ollama | model: qwen2.5-coder-7b | Ctrl+H: help
func (s *StatusBar) renderLine() string {
	return fmt.Sprintf(
		"%s │ latency: %dms │ cost: $%.4f │ llm: %s │ model: %s │ Ctrl+H: help",
		s.mode,
		s.telemetry.LatencyMS(),
		s.telemetry.SessionCost,
		s.telemetry.Backend,
		s.telemetry.ModelName,
	)
}
