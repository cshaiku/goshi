package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cshaiku/goshi/internal/audit"
)

// AuditPanel displays audit log events
type AuditPanel struct {
	width    int
	height   int
	viewport viewport.Model
	ready    bool

	// Data
	events   []audit.Event
	filePath string
}

// NewAuditPanel creates a new audit panel
func NewAuditPanel(filePath string) *AuditPanel {
	vp := viewport.New(30, 20)
	panel := &AuditPanel{
		viewport: vp,
		ready:    false,
		filePath: filePath,
		events:   []audit.Event{},
	}

	// Load events from file
	panel.loadEvents()

	return panel
}

// loadEvents reads events from the audit log file
func (p *AuditPanel) loadEvents() {
	if p.filePath == "" {
		return
	}

	file, err := os.Open(p.filePath)
	if err != nil {
		return // File might not exist yet
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		var event audit.Event
		if err := json.Unmarshal(line, &event); err != nil {
			continue // Skip malformed lines
		}
		p.events = append(p.events, event)
	}
}

// SetSize updates the panel dimensions
func (p *AuditPanel) SetSize(width, height int) {
	p.width = width
	p.height = height

	// Update viewport size
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
func (p *AuditPanel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return cmd
}

// Refresh reloads events from the file (call this periodically to see new events)
func (p *AuditPanel) Refresh() {
	p.events = []audit.Event{}
	p.loadEvents()
}

// Render returns the audit panel content
func (p *AuditPanel) Render() string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	// Build content
	content := p.renderHeader()

	if len(p.events) == 0 {
		content += "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("(no audit events yet)")
	} else {
		content += "\n\n" + p.renderEvents()
	}

	// Update viewport content if ready
	if p.ready {
		p.viewport.SetContent(content)
	}

	// Get scrollable viewport view
	viewportContent := p.viewport.View()

	// Apply border
	contentWidth := p.width - 4
	if contentWidth < 10 {
		contentWidth = 10
	}

	styled := borderStyle.Width(contentWidth).Render(viewportContent)
	return styled
}

func (p *AuditPanel) renderHeader() string {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")).
		Bold(true)

	countStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	return headerStyle.Render("═══ AUDIT LOG ═══") +
		"  " +
		countStyle.Render(fmt.Sprintf("(%d events)", len(p.events)))
}

func (p *AuditPanel) renderEvents() string {
	var output string

	// Show last N events (most recent first for better UX)
	startIdx := len(p.events) - 20
	if startIdx < 0 {
		startIdx = 0
	}

	// Render in reverse order (newest first)
	for i := len(p.events) - 1; i >= startIdx; i-- {
		event := p.events[i]
		output += p.formatEvent(event) + "\n"
	}

	return output
}

func (p *AuditPanel) formatEvent(e audit.Event) string {
	typeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14"))

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")) // Green for OK

	if e.Status == audit.StatusWarn {
		statusStyle = statusStyle.Foreground(lipgloss.Color("11")) // Yellow
	} else if e.Status == audit.StatusError {
		statusStyle = statusStyle.Foreground(lipgloss.Color("9")) // Red
	}

	timeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	msgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	timestamp := e.Timestamp.Format("15:04:05")
	timeStr := timeStyle.Render(timestamp)
	typeStr := typeStyle.Render(fmt.Sprintf("[%-12s]", e.Type))
	statusStr := statusStyle.Render(fmt.Sprintf("%-5s", e.Status))
	msgStr := msgStyle.Render(e.Message)

	line := fmt.Sprintf("%s %s %s %s", timeStr, typeStr, statusStr, msgStr)

	// Add action and details if available
	if e.Action != "" {
		actionStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("246"))
		line += "\n" + actionStyle.Render(fmt.Sprintf("  → %s", e.Action))
	}

	if len(e.Details) > 0 {
		detailStr := ""
		for k, v := range e.Details {
			if v == nil {
				continue
			}
			detailStr += fmt.Sprintf(" %s=%v", k, v)
		}
		if detailStr != "" {
			detailStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("242"))
			line += "\n" + detailStyle.Render(fmt.Sprintf("  %s", detailStr))
		}
	}

	return line
}
