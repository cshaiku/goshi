package openai

import (
	"fmt"
	"sync"
	"time"
)

// ModelPricing defines the cost per 1M tokens for OpenAI models
// Prices are in USD and should be updated periodically
// Source: https://openai.com/pricing (as of Feb 2026)
var ModelPricing = map[string]struct {
	InputPer1M  float64
	OutputPer1M float64
}{
	"gpt-4o":        {InputPer1M: 2.50, OutputPer1M: 10.00},
	"gpt-4o-mini":   {InputPer1M: 0.15, OutputPer1M: 0.60},
	"gpt-4-turbo":   {InputPer1M: 10.00, OutputPer1M: 30.00},
	"gpt-4":         {InputPer1M: 30.00, OutputPer1M: 60.00},
	"gpt-3.5-turbo": {InputPer1M: 0.50, OutputPer1M: 1.50},
}

// CostTracker tracks token usage and costs for a session
type CostTracker struct {
	mu                    sync.Mutex
	model                 string
	totalPromptTokens     int
	totalCompletionTokens int
	totalCost             float64
	requestCount          int
	startTime             time.Time
	warnThreshold         float64 // Warn when cost exceeds this (USD)
	maxCost               float64 // Fail when cost exceeds this (USD)
	warningIssued         bool
}

// NewCostTracker creates a new cost tracker
func NewCostTracker(model string, warnThreshold, maxCost float64) *CostTracker {
	return &CostTracker{
		model:         model,
		startTime:     time.Now(),
		warnThreshold: warnThreshold,
		maxCost:       maxCost,
	}
}

// RecordUsage records token usage and calculates cost
func (ct *CostTracker) RecordUsage(promptTokens, completionTokens int) (cost float64, warning string, err error) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	// Calculate cost for this request
	pricing, ok := ModelPricing[ct.model]
	if !ok {
		// Unknown model - use default pricing (gpt-4o)
		pricing = ModelPricing["gpt-4o"]
	}

	requestCost := (float64(promptTokens)/1_000_000)*pricing.InputPer1M +
		(float64(completionTokens)/1_000_000)*pricing.OutputPer1M

	// Update totals
	ct.totalPromptTokens += promptTokens
	ct.totalCompletionTokens += completionTokens
	ct.totalCost += requestCost
	ct.requestCount++

	// Check thresholds
	if ct.maxCost > 0 && ct.totalCost > ct.maxCost {
		return requestCost, "", fmt.Errorf("cost limit exceeded: $%.4f > $%.2f (session total: $%.4f)",
			ct.totalCost, ct.maxCost, ct.totalCost)
	}

	if ct.warnThreshold > 0 && ct.totalCost > ct.warnThreshold && !ct.warningIssued {
		warning = fmt.Sprintf("⚠️  Cost warning: Session total $%.4f exceeds threshold $%.2f",
			ct.totalCost, ct.warnThreshold)
		ct.warningIssued = true
	}

	return requestCost, warning, nil
}

// EstimateCost estimates the cost for a given number of tokens
func (ct *CostTracker) EstimateCost(promptTokens, completionTokens int) float64 {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	pricing, ok := ModelPricing[ct.model]
	if !ok {
		pricing = ModelPricing["gpt-4o"]
	}

	return (float64(promptTokens)/1_000_000)*pricing.InputPer1M +
		(float64(completionTokens)/1_000_000)*pricing.OutputPer1M
}

// GetSummary returns a summary of usage and costs
func (ct *CostTracker) GetSummary() CostSummary {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	return CostSummary{
		Model:                 ct.model,
		TotalPromptTokens:     ct.totalPromptTokens,
		TotalCompletionTokens: ct.totalCompletionTokens,
		TotalTokens:           ct.totalPromptTokens + ct.totalCompletionTokens,
		TotalCost:             ct.totalCost,
		RequestCount:          ct.requestCount,
		Duration:              time.Since(ct.startTime),
		AverageCostPerRequest: ct.totalCost / float64(ct.requestCount),
	}
}

// Reset resets the cost tracker
func (ct *CostTracker) Reset() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.totalPromptTokens = 0
	ct.totalCompletionTokens = 0
	ct.totalCost = 0
	ct.requestCount = 0
	ct.startTime = time.Now()
	ct.warningIssued = false
}

// CostSummary provides a summary of costs and usage
type CostSummary struct {
	Model                 string
	TotalPromptTokens     int
	TotalCompletionTokens int
	TotalTokens           int
	TotalCost             float64
	RequestCount          int
	Duration              time.Duration
	AverageCostPerRequest float64
}

// String returns a formatted string representation
func (cs CostSummary) String() string {
	return fmt.Sprintf(
		"[OpenAI Cost] Model: %s | Requests: %d | Tokens: %d (prompt: %d, completion: %d) | Cost: $%.4f (avg: $%.4f/req) | Duration: %s",
		cs.Model,
		cs.RequestCount,
		cs.TotalTokens,
		cs.TotalPromptTokens,
		cs.TotalCompletionTokens,
		cs.TotalCost,
		cs.AverageCostPerRequest,
		cs.Duration.Round(time.Second),
	)
}
