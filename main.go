package main

import (
	"github.com/cshaiku/goshi/internal/toolbridge"
	"github.com/cshaiku/goshi/internal/version"
	"log"
	"os"

	"github.com/cshaiku/goshi/internal/cli"
	"github.com/cshaiku/goshi/internal/llm"
	"github.com/cshaiku/goshi/internal/selfmodel"
)

func main() {
	if handled, code := toolbridge.Handle(os.Args[1:], "goshi", version.Version, os.Stdout, os.Stderr); handled {
		os.Exit(code)
	}
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
