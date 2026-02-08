package selfmodel

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

type greetingDoc struct {
	HumanGreeting string `yaml:"human_greeting"`
}

// ExtractHumanGreeting attempts to extract the human-facing greeting.
// Failure is non-fatal: returns empty string on any error.
func ExtractHumanGreeting(raw string) string {
	var doc greetingDoc

	dec := yaml.NewDecoder(bytes.NewReader([]byte(raw)))
	dec.KnownFields(false)

	if err := dec.Decode(&doc); err != nil {
		return ""
	}

	return doc.HumanGreeting
}

// SELF MODEL LAW INDEX
type lawsDoc struct {
	PrimaryLaws []string `yaml:"primary_laws"`
}

func ExtractPrimaryLaws(raw string) []string {
	var doc lawsDoc
	if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
		return nil
	}
	return doc.PrimaryLaws
}
