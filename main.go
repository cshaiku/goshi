package main

import (
	"log"

	"github.com/cshaiku/goshi/internal/cli"
	"github.com/cshaiku/goshi/internal/llm"
	"github.com/cshaiku/goshi/internal/selfmodel"
)

func main() {
	sm, err := selfmodel.Load("")
	if err != nil {
		log.Fatalf("startup aborted: %v", err)
	}

	systemPrompt, err := llm.NewSystemPrompt(sm.Raw)
	if err != nil {
		log.Fatalf("startup aborted: %v", err)
	}

	rt := &cli.Runtime{
		SystemPrompt: systemPrompt,
	}

	cli.Execute(rt)
}
