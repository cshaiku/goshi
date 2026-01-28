package selfmodel

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultPath = "grokgo.self.model.yaml"

func Load(path string) (*Model, error) {
	if path == "" {
		path = DefaultPath
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("self-model not found: %w", err)
	}

	var m Model
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("invalid self-model yaml: %w", err)
	}

	if m.Model.ModelVersion == "" {
		return nil, fmt.Errorf("self-model missing model.model_version")
	}

	if m.Application.Name == "" {
		return nil, fmt.Errorf("self-model missing application.name")
	}

	return &m, nil
}
