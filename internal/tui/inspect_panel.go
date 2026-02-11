package tui

import (
	"crypto/sha256"
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InspectPanel renders the right-side inspect panel with all 4 sections
type InspectPanel struct {
	width  int
	height int

	// Scrolling support
	viewport viewport.Model
	ready    bool

	// Data sources
	telemetry    *Telemetry
	lawsCount    int
	constCount   int
	guardrailsOn bool
	capabilities *Capabilities
}

// Capabilities represents system capabilities state
type Capabilities struct {
	ToolsEnabled      bool
	FilesystemAllowed bool
	FilesystemStatus  string // "allowed", "denied", "read-only"
	NetworkAllowed    bool
	NetworkStatus     string // "allowed", "denied", "restricted"
}

// NewInspectPanel creates a new inspect panel
func NewInspectPanel(telemetry *Telemetry) *InspectPanel {
	vp := viewport.New(30, 20)
	return &InspectPanel{
		telemetry: telemetry,
		viewport:  vp,
		ready:     false,
		capabilities: &Capabilities{
			ToolsEnabled:      true,
			FilesystemAllowed: false,
			FilesystemStatus:  "denied",
			NetworkAllowed:    false,
			NetworkStatus:     "denied",
		},
	}
}

// SetSize updates the panel dimensions
func (p *InspectPanel) SetSize(width, height int) {
	p.width = width
	p.height = height

	// Update viewport size (account for border and padding)
	contentWidth := width - 4
	contentHeight := height - 2
	if contentWidth < 10 {
		contentWidth = 10
	}
	if contentHeight < 5 {
		contentHeight = 5
	}

	p.viewport.Width = contentWidth
	p.viewport.Height = contentHeight
	p.ready = true
}

// Update handles viewport scrolling messages
func (p *InspectPanel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return cmd
}

// UpdateMetrics updates law and constraint counts
func (p *InspectPanel) UpdateMetrics(laws, constraints int) {
	p.lawsCount = laws
	p.constCount = constraints
}

// UpdateCapabilities updates the capabilities state
func (p *InspectPanel) UpdateCapabilities(caps *Capabilities) {
	if caps != nil {
		p.capabilities = caps
	}
}

// SetGuardrails sets the guardrail status
func (p *InspectPanel) SetGuardrails(enabled bool) {
	p.guardrailsOn = enabled
}

// Render returns the inspect panel content with all 4 sections
func (p *InspectPanel) Render(systemPrompt string) string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	// Render all sections as viewport content
	content := p.renderHeader() + "\n\n" +
		p.renderMemorySection() + "\n\n" +
		p.renderPromptInfoSection(systemPrompt) + "\n\n" +
		p.renderGuardrailsSection() + "\n\n" +
		p.renderCapabilitiesSection()

	// Update viewport content if ready
	if p.ready {
		p.viewport.SetContent(content)
	}

	// Get scrollable viewport view
	viewportContent := p.viewport.View()

	// Apply border to viewport
	contentWidth := p.width - 4
	if contentWidth < 10 {
		contentWidth = 10
	}

	styled := borderStyle.Width(contentWidth).Render(viewportContent)
	return styled
}

func (p *InspectPanel) renderHeader() string {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true)

	return headerStyle.Render("═══ INSPECT ═══")
}

func (p *InspectPanel) renderMemorySection() string {
	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	// Calculate memory usage percentage
	memPercent := 0.0
	if p.telemetry.MemoryMax > 0 {
		memPercent = (float64(p.telemetry.MemoryEntries) / float64(p.telemetry.MemoryMax)) * 100
	}

	// Memory bar
	barWidth := 20
	filled := int((memPercent / 100.0) * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	return sectionStyle.Render("MEMORY") + "\n" +
		valueStyle.Render(fmt.Sprintf("Entries: %d/%d", p.telemetry.MemoryEntries, p.telemetry.MemoryMax)) + "\n" +
		dimStyle.Render(bar) + " " + valueStyle.Render(fmt.Sprintf("%.0f%%", memPercent)) + "\n" +
		dimStyle.Render("Scope: ") + valueStyle.Render("session")
}

func (p *InspectPanel) renderPromptInfoSection(systemPrompt string) string {
	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	// Calculate policy hash from system prompt
	hash := sha256.Sum256([]byte(systemPrompt))
	policyHash := fmt.Sprintf("%X", hash[:3]) // First 6 hex chars

	return sectionStyle.Render("PROMPT INFO") + "\n" +
		dimStyle.Render("Policy Hash: ") + valueStyle.Render(policyHash) + "\n" +
		dimStyle.Render("Temperature: ") + valueStyle.Render(fmt.Sprintf("%.1f", p.telemetry.Temperature))
}

func (p *InspectPanel) renderGuardrailsSection() string {
	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	modeStyle := valueStyle
	modeText := "OFF"
	if p.guardrailsOn {
		modeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)
		modeText = "ON"
	}

	return sectionStyle.Render("GUARDRAILS") + "\n" +
		dimStyle.Render("Mode: ") + modeStyle.Render(modeText) + "\n" +
		dimStyle.Render("Laws: ") + valueStyle.Render(fmt.Sprintf("%d", p.lawsCount)) + "\n" +
		dimStyle.Render("Constraints: ") + valueStyle.Render(fmt.Sprintf("%d", p.constCount))
}

func (p *InspectPanel) renderCapabilitiesSection() string {
	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")).
		Bold(true)

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	enabledStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10"))

	deniedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("9"))

	// Tools status
	toolsStatus := deniedStyle.Render("disabled")
	if p.capabilities.ToolsEnabled {
		toolsStatus = enabledStyle.Render("enabled")
	}

	// Filesystem status
	fsStatus := deniedStyle.Render(p.capabilities.FilesystemStatus)
	if p.capabilities.FilesystemAllowed {
		fsStatus = enabledStyle.Render(p.capabilities.FilesystemStatus)
	}

	// Network status
	netStatus := deniedStyle.Render(p.capabilities.NetworkStatus)
	if p.capabilities.NetworkAllowed {
		netStatus = enabledStyle.Render(p.capabilities.NetworkStatus)
	}

	return sectionStyle.Render("CAPABILITIES") + "\n" +
		dimStyle.Render("Tools: ") + toolsStatus + "\n" +
		dimStyle.Render("Filesystem: ") + fsStatus + "\n" +
		dimStyle.Render("Network: ") + netStatus
}
