package openai

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(5, 30*time.Second)

	if cb.maxFailures != 5 {
		t.Errorf("expected maxFailures 5, got %d", cb.maxFailures)
	}
	if cb.timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", cb.timeout)
	}
	if cb.state != StateClosed {
		t.Errorf("expected initial state closed, got %v", cb.state)
	}
}

func TestCircuitBreaker_InitiallyAllowsRequests(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)

	if !cb.AllowRequest() {
		t.Error("circuit breaker should allow requests when closed")
	}
}

func TestCircuitBreaker_OpensAfterMaxFailures(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)

	// Record 2 failures - should stay closed
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.GetState() != StateClosed {
		t.Error("circuit should remain closed after 2 failures (max is 3)")
	}

	if !cb.AllowRequest() {
		t.Error("circuit should still allow requests")
	}

	// Third failure should open circuit
	cb.RecordFailure()

	if cb.GetState() != StateOpen {
		t.Error("circuit should open after 3 failures")
	}

	if cb.AllowRequest() {
		t.Error("circuit should not allow requests when open")
	}
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.GetState() != StateOpen {
		t.Error("circuit should be open")
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Next request should transition to half-open
	if !cb.AllowRequest() {
		t.Error("circuit should allow request after timeout, transitioning to half-open")
	}

	if cb.GetState() != StateHalfOpen {
		t.Errorf("circuit should be half-open, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_HalfOpenLimitsRequests(t *testing.T) {
	cb := NewCircuitBreaker(2, 10*time.Millisecond)

	// Open circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for half-open
	time.Sleep(15 * time.Millisecond)
	cb.AllowRequest() // Transition to half-open

	// Should allow limited requests (3 by default)
	if !cb.AllowRequest() {
		t.Error("half-open should allow request 1")
	}
	if !cb.AllowRequest() {
		t.Error("half-open should allow request 2")
	}
	if !cb.AllowRequest() {
		t.Error("half-open should allow request 3")
	}

	// Fourth request should be denied
	if cb.AllowRequest() {
		t.Error("half-open should not allow more than 3 requests")
	}
}

func TestCircuitBreaker_RecoveryToClosedState(t *testing.T) {
	cb := NewCircuitBreaker(2, 10*time.Millisecond)

	// Open circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Transition to half-open
	time.Sleep(15 * time.Millisecond)
	cb.AllowRequest()

	if cb.GetState() != StateHalfOpen {
		t.Error("should be half-open")
	}

	// Record 3 successes (halfOpenMaxReqs = 3)
	cb.RecordSuccess()
	cb.RecordSuccess()
	cb.RecordSuccess()

	// Should close circuit
	if cb.GetState() != StateClosed {
		t.Errorf("circuit should close after successful recovery, got %v", cb.GetState())
	}

	// Should allow requests freely again
	if !cb.AllowRequest() {
		t.Error("closed circuit should allow requests")
	}
}

func TestCircuitBreaker_FailureInHalfOpenReopens(t *testing.T) {
	cb := NewCircuitBreaker(2, 10*time.Millisecond)

	// Open circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Transition to half-open
	time.Sleep(15 * time.Millisecond)
	cb.AllowRequest()

	if cb.GetState() != StateHalfOpen {
		t.Error("should be half-open")
	}

	// Failure in half-open should reopen
	cb.RecordFailure()

	if cb.GetState() != StateOpen {
		t.Errorf("circuit should reopen on half-open failure, got %v", cb.GetState())
	}

	// Should not allow requests
	if cb.AllowRequest() {
		t.Error("reopened circuit should not allow requests")
	}
}

func TestCircuitBreaker_SuccessResetsFailureCount(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)

	// Record some failures
	cb.RecordFailure()
	cb.RecordFailure()

	stats := cb.GetStats()
	if stats.Failures != 2 {
		t.Errorf("expected 2 failures, got %d", stats.Failures)
	}

	// Success should reset count
	cb.RecordSuccess()

	stats = cb.GetStats()
	if stats.Failures != 0 {
		t.Errorf("success should reset failure count, got %d", stats.Failures)
	}

	// Circuit should still be closed
	if cb.GetState() != StateClosed {
		t.Error("circuit should remain closed")
	}
}

func TestCircuitBreaker_CallFunctionSuccess(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)

	called := false
	err := cb.Call(func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !called {
		t.Error("function should have been called")
	}

	stats := cb.GetStats()
	if stats.ConsecutiveSuccesses != 1 {
		t.Errorf("expected 1 success, got %d", stats.ConsecutiveSuccesses)
	}
}

func TestCircuitBreaker_CallFunctionFailure(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)

	expectedErr := errors.New("test error")
	err := cb.Call(func() error {
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("expected error to be returned, got %v", err)
	}

	stats := cb.GetStats()
	if stats.Failures != 1 {
		t.Errorf("expected 1 failure, got %d", stats.Failures)
	}
}

func TestCircuitBreaker_CallBlockedWhenOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, 10*time.Second)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	called := false
	err := cb.Call(func() error {
		called = true
		return nil
	})

	if err == nil {
		t.Error("expected error when circuit is open")
	}

	if called {
		t.Error("function should not be called when circuit is open")
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(2, 10*time.Second)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.GetState() != StateOpen {
		t.Error("circuit should be open")
	}

	// Reset
	cb.Reset()

	if cb.GetState() != StateClosed {
		t.Error("reset should close the circuit")
	}

	stats := cb.GetStats()
	if stats.Failures != 0 {
		t.Error("reset should clear failures")
	}
	if stats.ConsecutiveSuccesses != 0 {
		t.Error("reset should clear successes")
	}

	// Should allow requests
	if !cb.AllowRequest() {
		t.Error("reset circuit should allow requests")
	}
}

func TestCircuitBreaker_GetStats(t *testing.T) {
	cb := NewCircuitBreaker(5, 30*time.Second)

	// Record some activity
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess()

	stats := cb.GetStats()

	if stats.State != StateClosed {
		t.Errorf("expected state closed, got %v", stats.State)
	}
	if stats.Failures != 0 { // Success should reset
		t.Errorf("expected 0 failures after success, got %d", stats.Failures)
	}
	if stats.ConsecutiveSuccesses != 1 {
		t.Errorf("expected 1 success, got %d", stats.ConsecutiveSuccesses)
	}
}

func TestCircuitBreaker_TimeUntilHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Check time until half-open is approximately the timeout
	stats := cb.GetStats()
	if stats.TimeUntilHalfOpen < 90*time.Millisecond || stats.TimeUntilHalfOpen > 110*time.Millisecond {
		t.Errorf("expected ~100ms until half-open, got %v", stats.TimeUntilHalfOpen)
	}

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Time should decrease
	stats = cb.GetStats()
	if stats.TimeUntilHalfOpen < 40*time.Millisecond || stats.TimeUntilHalfOpen > 60*time.Millisecond {
		t.Errorf("expected ~50ms until half-open, got %v", stats.TimeUntilHalfOpen)
	}
}

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("state %v: expected %q, got %q", tt.state, tt.expected, got)
		}
	}
}

func TestCircuitBreakerStats_String(t *testing.T) {
	stats := CircuitBreakerStats{
		State:                StateClosed,
		Failures:             2,
		ConsecutiveSuccesses: 5,
	}

	str := stats.String()
	if str == "" {
		t.Error("stats string should not be empty")
	}

	// Should contain key information
	if !strings.Contains(str, "closed") {
		t.Error("stats string should contain state")
	}
}

func TestCircuitBreakerStats_StringWithTimeout(t *testing.T) {
	stats := CircuitBreakerStats{
		State:             StateOpen,
		Failures:          5,
		TimeUntilHalfOpen: 25 * time.Second,
	}

	str := stats.String()
	if !strings.Contains(str, "open") {
		t.Error("stats string should contain open state")
	}
	if !strings.Contains(str, "retry in") {
		t.Error("stats string should contain retry time for open state")
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(10, 1*time.Second)

	// Simulate concurrent operations
	done := make(chan bool)
	for i := 0; i < 20; i++ {
		go func(id int) {
			if id%2 == 0 {
				cb.RecordSuccess()
			} else {
				cb.RecordFailure()
			}
			cb.AllowRequest()
			cb.GetStats()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Should not panic and state should be consistent
	stats := cb.GetStats()
	if stats.Failures < 0 || stats.ConsecutiveSuccesses < 0 {
		t.Error("concurrent access caused inconsistent state")
	}
}
