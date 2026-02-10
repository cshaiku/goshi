package openai

import (
	"fmt"
	"sync"
	"time"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	StateClosed   CircuitState = iota // Normal operation
	StateOpen                         // Circuit is open, failing fast
	StateHalfOpen                     // Testing if service recovered
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern for OpenAI API
type CircuitBreaker struct {
	mu sync.Mutex

	// Configuration
	maxFailures     int           // Failures before opening circuit
	timeout         time.Duration // How long to wait before half-open
	halfOpenMaxReqs int           // Max requests to test in half-open state

	// State
	state                CircuitState
	failures             int
	consecutiveSuccesses int
	lastFailureTime      time.Time
	halfOpenRequests     int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:     maxFailures,
		timeout:         timeout,
		halfOpenMaxReqs: 3, // Test with 3 requests in half-open
		state:           StateClosed,
	}
}

// Call attempts to execute the function through the circuit breaker
func (cb *CircuitBreaker) Call(fn func() error) error {
	if !cb.AllowRequest() {
		return fmt.Errorf("circuit breaker is open: too many failures (last failure: %s ago)",
			time.Since(cb.lastFailureTime).Round(time.Second))
	}

	err := fn()

	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}

// AllowRequest checks if a request should be allowed
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if timeout has elapsed
		if time.Since(cb.lastFailureTime) > cb.timeout {
			// Transition to half-open to test
			cb.state = StateHalfOpen
			cb.halfOpenRequests = 0
			return true
		}
		return false

	case StateHalfOpen:
		// Allow limited requests in half-open state
		if cb.halfOpenRequests < cb.halfOpenMaxReqs {
			cb.halfOpenRequests++
			return true
		}
		return false

	default:
		return false
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveSuccesses++

	switch cb.state {
	case StateHalfOpen:
		// If we had enough successful requests in half-open, close the circuit
		if cb.consecutiveSuccesses >= cb.halfOpenMaxReqs {
			cb.state = StateClosed
			cb.failures = 0
			cb.halfOpenRequests = 0
			fmt.Printf("[OpenAI Circuit Breaker] Recovered: circuit closed after %d successful requests\n",
				cb.consecutiveSuccesses)
		}

	case StateClosed:
		// Reset failure count on success
		if cb.failures > 0 {
			cb.failures = 0
		}
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.consecutiveSuccesses = 0
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.maxFailures {
			cb.state = StateOpen
			fmt.Printf("[OpenAI Circuit Breaker] Opened: %d consecutive failures (cooldown: %s)\n",
				cb.failures, cb.timeout)
		}

	case StateHalfOpen:
		// Any failure in half-open goes back to open
		cb.state = StateOpen
		cb.halfOpenRequests = 0
		fmt.Printf("[OpenAI Circuit Breaker] Reopened: failure during recovery test\n")
	}
}

// GetState returns the current circuit state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return CircuitBreakerStats{
		State:                cb.state,
		Failures:             cb.failures,
		ConsecutiveSuccesses: cb.consecutiveSuccesses,
		LastFailureTime:      cb.lastFailureTime,
		TimeUntilHalfOpen:    cb.getTimeUntilHalfOpen(),
	}
}

// getTimeUntilHalfOpen calculates time until circuit can test again (must hold lock)
func (cb *CircuitBreaker) getTimeUntilHalfOpen() time.Duration {
	if cb.state != StateOpen {
		return 0
	}

	elapsed := time.Since(cb.lastFailureTime)
	if elapsed >= cb.timeout {
		return 0
	}

	return cb.timeout - elapsed
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.consecutiveSuccesses = 0
	cb.halfOpenRequests = 0
	fmt.Printf("[OpenAI Circuit Breaker] Manually reset to closed state\n")
}

// CircuitBreakerStats provides statistics about the circuit breaker
type CircuitBreakerStats struct {
	State                CircuitState
	Failures             int
	ConsecutiveSuccesses int
	LastFailureTime      time.Time
	TimeUntilHalfOpen    time.Duration
}

// String returns a formatted string representation
func (cbs CircuitBreakerStats) String() string {
	if cbs.State == StateOpen && cbs.TimeUntilHalfOpen > 0 {
		return fmt.Sprintf("Circuit: %s (failures: %d, retry in: %s)",
			cbs.State, cbs.Failures, cbs.TimeUntilHalfOpen.Round(time.Second))
	}
	return fmt.Sprintf("Circuit: %s (failures: %d, successes: %d)",
		cbs.State, cbs.Failures, cbs.ConsecutiveSuccesses)
}
