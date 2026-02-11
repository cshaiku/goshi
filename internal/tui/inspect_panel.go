package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// InspectPanel renders the right-side inspect panel
type InspectPanel struct {
	width  int
	height int
}

// NewInspectPanel creates a new inspect panel
func NewInspectPanel() *InspectPanel {
	return &InspectPanel{}
}

// SetSize updates the panel dimensions
func (p *InspectPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// Render returns the inspect panel content (stub for Phase 1)
func (p *InspectPanel) Render() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(p.width - 2).
		Height(p.height - 2).
		Padding(1)

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true)

	noteStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)

	content := headerStyle.Render("INSPECT PANEL") + "\n\n" +
		noteStyle.Render("Coming in Phase 2") + "\n\n" +
		"Sections:\n" +
		"• Memory\n" +
		"• Prompt Info\n" +
		"• Guardrails\n" +
		"• Capabilities"

	return style.Render(content)
}
