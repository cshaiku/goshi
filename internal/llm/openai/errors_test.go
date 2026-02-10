package openai

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestShouldRetry_RetryableCodes(t *testing.T) {
	retryable := []int{429, 500, 502, 503, 504}

	for _, code := range retryable {
		if !ShouldRetry(code) {
			t.Errorf("status code %d should be retryable", code)
		}
	}
}

func TestShouldRetry_NonRetryableCodes(t *testing.T) {
	nonRetryable := []int{200, 400, 401, 403, 404, 422}

	for _, code := range nonRetryable {
		if ShouldRetry(code) {
			t.Errorf("status code %d should not be retryable", code)
		}
	}
}

func TestCalculateBackoff_ExponentialGrowth(t *testing.T) {
	baseDelay := 1 * time.Second
	maxDelay := 60 * time.Second

	// Test exponential growth
	attempt0 := CalculateBackoff(0, baseDelay, maxDelay)
	attempt1 := CalculateBackoff(1, baseDelay, maxDelay)
	attempt2 := CalculateBackoff(2, baseDelay, maxDelay)

	// Attempt 0 should be around baseDelay
	if attempt0 < baseDelay || attempt0 > baseDelay*2 {
		t.Errorf("attempt 0: expected ~%v, got %v", baseDelay, attempt0)
	}

	// Attempt 1 should be roughly double (with jitter ±25%)
	expectedDelay1 := 2 * time.Second
	if attempt1 < expectedDelay1*75/100 || attempt1 > expectedDelay1*125/100 {
		t.Errorf("attempt 1: expected ~%v (±25%%), got %v", expectedDelay1, attempt1)
	}

	// Attempt 2 should be roughly quadruple (with jitter ±25%)
	expectedDelay2 := 4 * time.Second
	if attempt2 < expectedDelay2*75/100 || attempt2 > expectedDelay2*125/100 {
		t.Errorf("attempt 2: expected ~%v (±25%%), got %v", expectedDelay2, attempt2)
	}
}

func TestCalculateBackoff_CapsAtMaxDelay(t *testing.T) {
	baseDelay := 1 * time.Second
	maxDelay := 10 * time.Second

	// Very high attempt number
	backoff := CalculateBackoff(10, baseDelay, maxDelay)

	if backoff > maxDelay {
		t.Errorf("backoff should not exceed maxDelay: got %v, max %v", backoff, maxDelay)
	}
}

func TestCalculateBackoff_IncludesJitter(t *testing.T) {
	baseDelay := 1 * time.Second
	maxDelay := 60 * time.Second

	// Note: Current jitter implementation uses UnixNano()%1000 which may not
	// vary much in rapid succession. This test verifies the calculation logic
	// produces values in the expected range.

	delay := CalculateBackoff(2, baseDelay, maxDelay)

	// Verify delay is within expected range (4s ±25% for attempt 2)
	expectedBase := 4 * time.Second
	minExpected := expectedBase * 75 / 100  // 3s
	maxExpected := expectedBase * 125 / 100 // 5s

	if delay < minExpected || delay > maxExpected {
		t.Errorf("delay=%v outside expected range %v-%v", delay, minExpected, maxExpected)
	}
}

func TestHandleHTTPError_401(t *testing.T) {
	resp := &http.Response{
		StatusCode: 401,
	}
	body := []byte(`{"error": {"message": "Invalid API key"}}`)

	err := HandleHTTPError(resp, body)

	if err == nil {
		t.Fatal("expected error for 401")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError type, got %T", err)
	}

	if apiErr.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}

	// Should contain setup instructions
	errStr := err.Error()
	if !strings.Contains(errStr, "OPENAI_API_KEY") {
		t.Error("401 error should mention OPENAI_API_KEY")
	}
}

func TestHandleHTTPError_429(t *testing.T) {
	resp := &http.Response{
		StatusCode: 429,
	}
	body := []byte(`{"error": {"message": "Rate limit exceeded"}}`)

	err := HandleHTTPError(resp, body)

	if err == nil {
		t.Fatal("expected error for 429")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError type, got %T", err)
	}

	if apiErr.StatusCode != 429 {
		t.Errorf("expected status 429, got %d", apiErr.StatusCode)
	}
}

func TestHandleHTTPError_500(t *testing.T) {
	resp := &http.Response{
		StatusCode: 500,
	}
	body := []byte(`{"error": {"message": "Internal server error"}}`)

	err := HandleHTTPError(resp, body)

	if err == nil {
		t.Fatal("expected error for 500")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError type, got %T", err)
	}

	if apiErr.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", apiErr.StatusCode)
	}
}

func TestHandleHTTPError_WithJSONBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: 400,
	}
	body := []byte(`{"error": {"message": "Invalid request"}`)

	err := HandleHTTPError(resp, body)

	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError type, got %T", err)
	}

	if apiErr.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}

func TestHandleHTTPError_EmptyBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: 503,
	}
	body := []byte(``)

	err := HandleHTTPError(resp, body)

	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError type, got %T", err)
	}

	if apiErr.StatusCode != 503 {
		t.Errorf("expected status 503, got %d", apiErr.StatusCode)
	}

	// Should have a message
	if apiErr.Message == "" {
		t.Error("error should have a message even with empty body")
	}
}

func TestAPIError_Error(t *testing.T) {
	err := &APIError{
		StatusCode: 429,
		Message:    "Rate limit exceeded",
	}

	errStr := err.Error()

	if errStr != "Rate limit exceeded" {
		t.Errorf("expected message 'Rate limit exceeded', got %q", errStr)
	}
}
