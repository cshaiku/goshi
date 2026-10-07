package tui

import (
	"crypto/sha256"
	"fmt"
	"strings"

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
	telemetry     *Telemetry
	lawsCount     int
	constCount    int
	guardrailsOn  bool
	dryRun        bool
	deterministic bool
	capabilities  *Capabilities
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

// UpdateToggles updates the input toggle states.
func (p *InspectPanel) UpdateToggles(dryRun, deterministic bool) {
	p.dryRun = dryRun
	p.deterministic = deterministic
}

// SetGuardrails sets the guardrail status
func (p *InspectPanel) SetGuardrails(enabled bool) {
	p.guardrailsOn = enabled
}

// Capabilities returns the current capabilities state.
func (p *InspectPanel) Capabilities() *Capabilities {
	return p.capabilities
}

// GuardrailsOn reports whether guardrails are enabled.
func (p *InspectPanel) GuardrailsOn() bool {
	return p.guardrailsOn
}

// Render returns the inspect panel content with all 4 sections
func (p *InspectPanel) Render(systemPrompt string) string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	// Render all sections as viewport content
	content := p.renderMemorySection() + "\n\n" +
		p.renderPromptInfoSection(systemPrompt) + "\n\n" +
		p.renderGuardrailsSection() + "\n\n" +
		p.renderTogglesSection() + "\n\n" +
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

	styled := borderStyle.Width(p.width).Height(p.height).Render(viewportContent)
	return styled
}

func (p *InspectPanel) contentWidth() int {
	width := p.viewport.Width
	if width <= 0 {
		width = p.width - 4
	}
	if width < 10 {
		width = 10
	}
	return width
}

func (p *InspectPanel) formatKV(label, value string, labelStyle, valueStyle lipgloss.Style) string {
	left := labelStyle.Render(label)
	right := valueStyle.Render(value)
	space := p.contentWidth() - lipgloss.Width(left) - lipgloss.Width(right)
	if space < 1 {
		space = 1
	}
	return left + strings.Repeat(" ", space) + right
}

func (p *InspectPanel) renderMemorySection() string {
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

	entriesLine := p.formatKV(
		"Entries",
		fmt.Sprintf("%d/%d", p.telemetry.MemoryEntries, p.telemetry.MemoryMax),
		dimStyle,
		valueStyle,
	)
	usageLine := p.formatKV(
		"Usage",
		fmt.Sprintf("%.0f%%", memPercent),
		dimStyle,
		valueStyle,
	)

	return entriesLine + "\n" + usageLine + "\n" + dimStyle.Render(bar)
}

func (p *InspectPanel) renderPromptInfoSection(systemPrompt string) string {
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	// Calculate policy hash from system prompt
	hash := sha256.Sum256([]byte(systemPrompt))
	policyHash := fmt.Sprintf("%X", hash[:3]) // First 6 hex chars

	return p.formatKV("Session", policyHash, dimStyle, valueStyle) + "\n" +
		p.formatKV("Temperature", fmt.Sprintf("%.1f", p.telemetry.Temperature), dimStyle, valueStyle)
}

func (p *InspectPanel) renderGuardrailsSection() string {
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

	return p.formatKV("Guard", modeText, dimStyle, modeStyle) + "\n" +
		p.formatKV("Laws", fmt.Sprintf("%d", p.lawsCount), dimStyle, valueStyle) + "\n" +
		p.formatKV("Constraints", fmt.Sprintf("%d", p.constCount), dimStyle, valueStyle)
}

func (p *InspectPanel) renderCapabilitiesSection() string {
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	enabledStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10"))

	deniedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("9"))

	// Tools status
	toolsValue := "disabled"
	toolsStyle := deniedStyle
	if p.capabilities.ToolsEnabled {
		toolsValue = "enabled"
		toolsStyle = enabledStyle
	}

	// Filesystem status
	fsValue := p.capabilities.FilesystemStatus
	fsStyle := deniedStyle
	if p.capabilities.FilesystemAllowed {
		fsStyle = enabledStyle
	}

	// Network status
	netValue := p.capabilities.NetworkStatus
	netStyle := deniedStyle
	if p.capabilities.NetworkAllowed {
		netStyle = enabledStyle
	}

	return p.formatKV("Tools", toolsValue, dimStyle, toolsStyle) + "\n" +
		p.formatKV("Filesystem", fsValue, dimStyle, fsStyle) + "\n" +
		p.formatKV("Network", netValue, dimStyle, netStyle)
}

func (p *InspectPanel) renderTogglesSection() string {
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	dryRunText := "OFF"
	if p.dryRun {
		dryRunText = "ON"
	}

	deterministicText := "OFF"
	if p.deterministic {
		deterministicText = "ON"
	}

	return p.formatKV("Dry Run", dryRunText, dimStyle, valueStyle) + "\n" +
		p.formatKV("Deterministic", deterministicText, dimStyle, valueStyle)
}
