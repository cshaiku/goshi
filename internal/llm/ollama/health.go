package ollama

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func CheckHealth(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		"http://127.0.0.1:11434/api/tags",
		nil,
	)
	if err != nil {
		return err
	}

	client := http.Client{
		Timeout: 500 * time.Millisecond,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama unhealthy: %s", resp.Status)
	}

	return nil
}
