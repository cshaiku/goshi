package tui

import (
	"time"
)

// Telemetry tracks real-time metrics for the TUI
type Telemetry struct {
	// Request metrics
	LastLatency    time.Duration
	AverageLatency time.Duration
	RequestCount   int64
	TotalLatency   time.Duration

	// Token tracking
	TokensUsed  int64
	TokensLimit int64

	// Cost tracking
	SessionCost float64

	// Memory tracking
	MemoryEntries int
	MemoryMax     int

	// Model info
	Temperature float64
	ModelName   string
	Backend     string

	// Status
	Status string // STAGED, ACTIVE, PENDING
}

// NewTelemetry creates a new telemetry tracker
func NewTelemetry() *Telemetry {
	return &Telemetry{
		TokensLimit: 16384, // Default context window
		MemoryMax:   128,   // Default memory capacity
		Temperature: 0.2,   // Default temperature
		Status:      "STAGED",
	}
}

// RecordRequest records metrics for a completed LLM request
func (t *Telemetry) RecordRequest(latency time.Duration, tokensUsed int, cost float64) {
	t.LastLatency = latency
	t.RequestCount++
	t.TotalLatency += latency
	t.AverageLatency = t.TotalLatency / time.Duration(t.RequestCount)
	t.TokensUsed += int64(tokensUsed)
	t.SessionCost += cost
}

// UpdateMemory updates memory usage
func (t *Telemetry) UpdateMemory(entries int) {
	t.MemoryEntries = entries
}

// UpdateStatus updates the system status
func (t *Telemetry) UpdateStatus(status string) {
	t.Status = status
}

// LatencyMS returns latency in milliseconds
func (t *Telemetry) LatencyMS() int64 {
	return t.LastLatency.Milliseconds()
}

// TokenUsagePercent returns token usage as a percentage
func (t *Telemetry) TokenUsagePercent() float64 {
	if t.TokensLimit == 0 {
		return 0
	}
	return (float64(t.TokensUsed) / float64(t.TokensLimit)) * 100
}
