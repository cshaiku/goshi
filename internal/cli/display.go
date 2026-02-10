package cli

// ANSI color codes for terminal output
// Extracted per SOLID principles to avoid magic strings
const (
	ColorReset  = "\033[0m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorRed    = "\033[31m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
)

// DisplayConfig holds UI formatting preferences
type DisplayConfig struct {
	EnableColors bool
}

// DefaultDisplayConfig returns sensible defaults
func DefaultDisplayConfig() *DisplayConfig {
	return &DisplayConfig{
		EnableColors: true,
	}
}

// Colorize wraps text with ANSI color codes if colors are enabled
func (d *DisplayConfig) Colorize(text, color string) string {
	if !d.EnableColors {
		return text
	}
	return color + text + ColorReset
}
