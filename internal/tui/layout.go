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
	TopBarHeight       int

	// State
	AuditPanelVisible bool

	// Split ratio (0.0 to 1.0)
	SplitRatio float64
}

// NewLayout creates a new layout with default split ratio
func NewLayout() *Layout {
	return &Layout{
		SplitRatio:      0.70, // Legacy: no longer used for layout.
		StatusBarHeight: 1,    // Single line for status bar
		InputHeight:     4,    // Input area height
		TopBarHeight:    2,    // Two-line top bar
	}
}

// Recalculate updates all dimensions based on terminal size
func (l *Layout) Recalculate(width, height int) {
	l.TerminalWidth = width
	l.TerminalHeight = height

	// Full-width output stream; sidebar removed.
	l.OutputStreamWidth = width
	l.InspectPanelWidth = 0

	// Calculate vertical dimensions
	// Reserve space for top bar, status bar, and input
	totalReserved := l.StatusBarHeight + l.InputHeight + l.TopBarHeight

	l.AuditPanelHeight = 0
	if height > totalReserved {
		l.OutputStreamHeight = height - totalReserved
	} else {
		l.OutputStreamHeight = 10 // minimum height
	}
}

// MinimumSize returns the minimum terminal dimensions required
func (l *Layout) MinimumSize() (width, height int) {
	return 80, 24
}
