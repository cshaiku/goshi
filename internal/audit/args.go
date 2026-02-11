package audit

import (
	"fmt"
	"strings"
)

func FormatToolArgs(args map[string]any, style string, redact bool) map[string]any {
	if args == nil {
		return nil
	}

	clean := sanitizeMap(args, redact)

	switch style {
	case "full":
		return clean
	case "long":
		return truncateMap(clean, 200, 20)
	case "short":
		return truncateMap(clean, 80, 5)
	case "summaries":
		return summarizeMap(clean)
	default:
		return summarizeMap(clean)
	}
}

func summarizeMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		switch v := value.(type) {
		case string:
			out[key] = truncateString(v, 40)
		case []any:
			out[key] = fmt.Sprintf("array[%d]", len(v))
		case map[string]any:
			out[key] = fmt.Sprintf("object[%d]", len(v))
		default:
			out[key] = v
		}
	}
	return out
}

func truncateMap(input map[string]any, maxString int, maxItems int) map[string]any {
	out := make(map[string]any, len(input))
	count := 0
	for key, value := range input {
		if maxItems > 0 && count >= maxItems {
			out["_truncated"] = fmt.Sprintf("%d+ keys", len(input))
			break
		}
		count++
		switch v := value.(type) {
		case string:
			out[key] = truncateString(v, maxString)
		case []any:
			out[key] = truncateSlice(v, maxString, maxItems)
		case map[string]any:
			out[key] = truncateMap(v, maxString, maxItems)
		default:
			out[key] = v
		}
	}
	return out
}

func truncateSlice(input []any, maxString int, maxItems int) []any {
	if maxItems > 0 && len(input) > maxItems {
		input = input[:maxItems]
	}
	out := make([]any, 0, len(input))
	for _, value := range input {
		switch v := value.(type) {
		case string:
			out = append(out, truncateString(v, maxString))
		case map[string]any:
			out = append(out, truncateMap(v, maxString, maxItems))
		default:
			out = append(out, v)
		}
	}
	if maxItems > 0 && len(input) == maxItems {
		out = append(out, "...")
	}
	return out
}

func truncateString(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max] + "..."
}

func sanitizeMap(input map[string]any, redact bool) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		if redact && shouldRedactKey(key) {
			out[key] = "***redacted***"
			continue
		}
		switch v := value.(type) {
		case map[string]any:
			out[key] = sanitizeMap(v, redact)
		case []any:
			out[key] = sanitizeSlice(v, redact)
		default:
			out[key] = v
		}
	}
	return out
}

func sanitizeSlice(input []any, redact bool) []any {
	out := make([]any, 0, len(input))
	for _, value := range input {
		switch v := value.(type) {
		case map[string]any:
			out = append(out, sanitizeMap(v, redact))
		case []any:
			out = append(out, sanitizeSlice(v, redact))
		case string:
			out = append(out, v)
		default:
			out = append(out, v)
		}
	}
	return out
}

func shouldRedactKey(key string) bool {
	lower := strings.ToLower(key)
	markers := []string{"token", "secret", "password", "apikey", "api_key", "authorization", "auth", "key"}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
