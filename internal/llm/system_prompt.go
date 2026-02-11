package llm

import (
	"fmt"
)

type SystemPrompt struct {
	raw string
}

func NewSystemPrompt(selfModelRaw string) (*SystemPrompt, error) {
	if selfModelRaw == "" {
		return nil, fmt.Errorf("system prompt requires non-empty self-model text")
	}

	return &SystemPrompt{
		raw: selfModelRaw,
	}, nil
}

func (s *SystemPrompt) Raw() string {
	return s.raw
}
