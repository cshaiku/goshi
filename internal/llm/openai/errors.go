package openai

import (
	"fmt"
	"math"
	"net/http"
	"time"
)

// APIError represents an error from the OpenAI API
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	return e.Message
}

// HandleHTTPError converts HTTP errors to user-friendly error messages
func HandleHTTPError(resp *http.Response, body []byte) error {
	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Body:       string(body),
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		apiErr.Message = "OpenAI API authentication failed (401)\n\nYour API key is invalid or expired.\nPlease check OPENAI_API_KEY environment variable.\n\nGet a new key at: https://platform.openai.com/api-keys"

	case http.StatusTooManyRequests:
		apiErr.Message = fmt.Sprintf("OpenAI API rate limit exceeded (429)\n\nYou've sent too many requests.\nPlease wait a moment and try again.\n\nError details: %s", string(body))

	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		apiErr.Message = fmt.Sprintf("OpenAI API server error (%d)\n\nOpenAI's servers are experiencing issues.\nPlease try again in a few moments.\n\nError details: %s", resp.StatusCode, string(body))

	default:
		apiErr.Message = fmt.Sprintf("OpenAI API error (%d): %s", resp.StatusCode, string(body))
	}

	return apiErr
}

// ShouldRetry determines if an error is retryable
func ShouldRetry(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests: // 429
		return true
	case http.StatusInternalServerError: // 500
		return true
	case http.StatusBadGateway: // 502
		return true
	case http.StatusServiceUnavailable: // 503
		return true
	case http.StatusGatewayTimeout: // 504
		return true
	default:
		return false
	}
}

// CalculateBackoff returns the backoff duration for retry attempt n (0-indexed)
// Uses exponential backoff with jitter
func CalculateBackoff(attempt int, minBackoff, maxBackoff time.Duration) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Exponential: min * 2^attempt
	backoff := float64(minBackoff) * math.Pow(2, float64(attempt))

	// Cap at maxBackoff
	if backoff > float64(maxBackoff) {
		backoff = float64(maxBackoff)
	}

	// Add jitter (±25%)
	jitterFactor := 2*float64(time.Now().UnixNano()%1000)/1000.0 - 1
	jitter := backoff * 0.25 * jitterFactor
	backoff += jitter

	// Cap again after adding jitter to ensure we never exceed maxBackoff
	if backoff > float64(maxBackoff) {
		backoff = float64(maxBackoff)
	}

	// Ensure minimum
	if backoff < float64(minBackoff) {
		backoff = float64(minBackoff)
	}

	return time.Duration(backoff)
}
