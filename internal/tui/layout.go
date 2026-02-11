package tui

// FocusRegion represents which part of the UI has focus
type FocusRegion int

const (
	FocusOutputStream FocusRegion = iota
	FocusInspectPanel
	FocusAuditPanel
	FocusInput
)

// Layout manages the three-region TUI layout calculations
type Layout struct {
	// Terminal dimensions
	TerminalWidth  int
	TerminalHeight int

	// Calculated dimensions
	OutputStreamWidth  int
	InspectPanelWidth  int
	OutputStreamHeight int
	AuditPanelHeight   int
	StatusBarHeight    int
	InputHeight        int

	// State
	AuditPanelVisible bool

	// Split ratio (0.0 to 1.0)
	SplitRatio float64
}

// NewLayout creates a new layout with default split ratio
func NewLayout() *Layout {
	return &Layout{
		SplitRatio:      0.70, // 70% output, 30% inspect panel
		StatusBarHeight: 2,    // Two lines for status bar
		InputHeight:     4,    // Input area height
	}
}

// Recalculate updates all dimensions based on terminal size
func (l *Layout) Recalculate(width, height int) {
	l.TerminalWidth = width
	l.TerminalHeight = height

	// Calculate horizontal split
	l.OutputStreamWidth = int(float64(width) * l.SplitRatio)
	l.InspectPanelWidth = width - l.OutputStreamWidth

	// Ensure minimum widths
	if l.OutputStreamWidth < 40 {
		l.OutputStreamWidth = 40
	}
	if l.InspectPanelWidth < 20 {
		l.InspectPanelWidth = 20
	}

	// Calculate vertical dimensions
	// Reserve space for status bar and input
	totalReserved := l.StatusBarHeight + l.InputHeight

	// If audit panel is visible, it gets ~1/3 of remaining height, streams get ~2/3
	if l.AuditPanelVisible {
		remainingHeight := height - totalReserved
		l.AuditPanelHeight = remainingHeight / 3
		l.OutputStreamHeight = remainingHeight - l.AuditPanelHeight
		if l.OutputStreamHeight < 10 {
			l.OutputStreamHeight = 10
		}
		if l.AuditPanelHeight < 5 {
			l.AuditPanelHeight = 5
		}
	} else {
		l.AuditPanelHeight = 0
		if height > totalReserved {
			l.OutputStreamHeight = height - totalReserved
		} else {
			l.OutputStreamHeight = 10 // minimum height
		}
	}
}

// MinimumSize returns the minimum terminal dimensions required
func (l *Layout) MinimumSize() (width, height int) {
	return 80, 24
}
