package openai

import (
	"strings"
	"testing"
	"time"
)

func TestNewCostTracker(t *testing.T) {
	ct := NewCostTracker("gpt-4o", 1.0, 10.0)

	if ct.model != "gpt-4o" {
		t.Errorf("expected model gpt-4o, got %s", ct.model)
	}
	if ct.warnThreshold != 1.0 {
		t.Errorf("expected warn threshold 1.0, got %.2f", ct.warnThreshold)
	}
	if ct.maxCost != 10.0 {
		t.Errorf("expected max cost 10.0, got %.2f", ct.maxCost)
	}
	if ct.totalCost != 0 {
		t.Error("expected initial total cost to be 0")
	}
}

func TestCostTracker_RecordUsage(t *testing.T) {
	ct := NewCostTracker("gpt-4o-mini", 0, 0) // No thresholds

	// Record usage: 1000 prompt tokens, 500 completion tokens
	cost, warning, err := ct.RecordUsage(1000, 500)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if warning != "" {
		t.Errorf("unexpected warning: %s", warning)
	}

	// gpt-4o-mini: $0.15/1M input, $0.60/1M output
	// Expected: (1000/1M * 0.15) + (500/1M * 0.60) = 0.00015 + 0.0003 = 0.00045
	expectedCost := 0.00045
	if cost < expectedCost-0.00001 || cost > expectedCost+0.00001 {
		t.Errorf("expected cost ~%.6f, got %.6f", expectedCost, cost)
	}

	summary := ct.GetSummary()
	if summary.TotalPromptTokens != 1000 {
		t.Errorf("expected 1000 prompt tokens, got %d", summary.TotalPromptTokens)
	}
	if summary.TotalCompletionTokens != 500 {
		t.Errorf("expected 500 completion tokens, got %d", summary.TotalCompletionTokens)
	}
	if summary.RequestCount != 1 {
		t.Errorf("expected 1 request, got %d", summary.RequestCount)
	}
}

func TestCostTracker_WarningThreshold(t *testing.T) {
	ct := NewCostTracker("gpt-4o-mini", 0.01, 0) // Warn at $0.01

	// Record enough usage to exceed warning threshold
	// 10,000 prompt + 5,000 completion = (10000/1M * 0.15) + (5000/1M * 0.60)
	// = 0.0015 + 0.003 = 0.0045
	_, warning1, _ := ct.RecordUsage(10000, 5000)
	if warning1 != "" {
		t.Error("should not warn on first request below threshold")
	}

	// Add more to exceed threshold
	// 10,000 + 5,000 more = 0.0045 + 0.0045 = 0.009 (still below)
	_, warning2, _ := ct.RecordUsage(10000, 5000)
	if warning2 != "" {
		t.Error("should not warn yet")
	}

	// One more push over the threshold
	_, warning3, _ := ct.RecordUsage(10000, 5000)
	if warning3 == "" {
		t.Error("expected warning when exceeding threshold")
	}

	// Should not warn again
	_, warning4, _ := ct.RecordUsage(1000, 500)
	if warning4 != "" {
		t.Error("warning should only fire once")
	}
}

func TestCostTracker_MaxCostLimit(t *testing.T) {
	ct := NewCostTracker("gpt-4o-mini", 0, 0.01) // Max $0.01

	// Record usage that exceeds max cost
	// Need to hit $0.01 with gpt-4o-mini ($0.15/1M input, $0.60/1M output)
	// Use 100,000 prompt + 50,000 completion
	// = (100000/1M * 0.15) + (50000/1M * 0.60) = 0.015 + 0.03 = 0.045
	_, _, err := ct.RecordUsage(100000, 50000)

	if err == nil {
		t.Error("expected error when exceeding max cost")
	}

	summary := ct.GetSummary()
	if summary.TotalCost <= 0.01 {
		t.Errorf("expected total cost > 0.01, got %.4f", summary.TotalCost)
	}
}

func TestCostTracker_EstimateCost(t *testing.T) {
	ct := NewCostTracker("gpt-4o", 0, 0)

	// Estimate for 1000 prompt + 500 completion
	// gpt-4o: $2.50/1M input, $10.00/1M output
	// = (1000/1M * 2.50) + (500/1M * 10.00) = 0.0025 + 0.005 = 0.0075
	estimated := ct.EstimateCost(1000, 500)

	expectedCost := 0.0075
	if estimated < expectedCost-0.0001 || estimated > expectedCost+0.0001 {
		t.Errorf("expected estimate ~%.4f, got %.4f", expectedCost, estimated)
	}

	// Should not affect totals
	summary := ct.GetSummary()
	if summary.TotalCost != 0 {
		t.Error("estimate should not affect total cost")
	}
}

func TestCostTracker_UnknownModel(t *testing.T) {
	ct := NewCostTracker("unknown-model", 0, 0)

	// Should fall back to gpt-4o pricing
	cost, _, _ := ct.RecordUsage(1000, 500)

	// gpt-4o default: $2.50/1M input, $10.00/1M output
	expectedCost := 0.0075
	if cost < expectedCost-0.0001 || cost > expectedCost+0.0001 {
		t.Errorf("expected default pricing ~%.4f, got %.4f", expectedCost, cost)
	}
}

func TestCostTracker_Reset(t *testing.T) {
	ct := NewCostTracker("gpt-4o-mini", 1.0, 10.0)

	// Record some usage
	ct.RecordUsage(1000, 500)
	ct.RecordUsage(2000, 1000)

	summary := ct.GetSummary()
	if summary.RequestCount != 2 {
		t.Errorf("expected 2 requests before reset, got %d", summary.RequestCount)
	}

	// Reset
	ct.Reset()

	summary = ct.GetSummary()
	if summary.TotalCost != 0 {
		t.Error("expected cost to be reset to 0")
	}
	if summary.RequestCount != 0 {
		t.Error("expected request count to be reset to 0")
	}
	if summary.TotalPromptTokens != 0 {
		t.Error("expected prompt tokens to be reset to 0")
	}
}

func TestCostTracker_ConcurrentAccess(t *testing.T) {
	ct := NewCostTracker("gpt-4o-mini", 0, 0)

	// Simulate concurrent requests
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			ct.RecordUsage(100, 50)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	summary := ct.GetSummary()
	if summary.RequestCount != 10 {
		t.Errorf("expected 10 requests in concurrent test, got %d", summary.RequestCount)
	}
	if summary.TotalPromptTokens != 1000 {
		t.Errorf("expected 1000 total prompt tokens, got %d", summary.TotalPromptTokens)
	}
}

func TestCostSummary_String(t *testing.T) {
	ct := NewCostTracker("gpt-4o", 0, 0)
	ct.RecordUsage(1000, 500)

	// Let some time pass for duration
	time.Sleep(10 * time.Millisecond)

	summary := ct.GetSummary()
	str := summary.String()

	if str == "" {
		t.Error("summary string should not be empty")
	}

	// Check it contains expected fields
	expectedSubstrings := []string{"gpt-4o", "Tokens:", "Cost:", "Requests:"}
	for _, substr := range expectedSubstrings {
		if !strings.Contains(str, substr) {
			t.Errorf("expected summary to contain %q", substr)
		}
	}
}

func TestModelPricing_AllModels(t *testing.T) {
	expectedModels := []string{
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4-turbo",
		"gpt-4",
		"gpt-3.5-turbo",
	}

	for _, model := range expectedModels {
		pricing, exists := ModelPricing[model]
		if !exists {
			t.Errorf("model %s not in pricing table", model)
			continue
		}

		if pricing.InputPer1M <= 0 {
			t.Errorf("model %s has invalid input price: %.2f", model, pricing.InputPer1M)
		}
		if pricing.OutputPer1M <= 0 {
			t.Errorf("model %s has invalid output price: %.2f", model, pricing.OutputPer1M)
		}
	}
}
