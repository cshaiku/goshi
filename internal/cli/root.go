package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var runtime *Runtime

var rootCmd = &cobra.Command{
	Use:   "goshi",
	Short: "Goshi is a local-first protective CLI agent",

	// This restores the original behavior:
	// running `goshi` immediately enters chat.
	Run: func(cmd *cobra.Command, args []string) {
		if runtime == nil || runtime.SystemPrompt == nil {
			fmt.Fprintln(os.Stderr, "fatal: system prompt not initialized")
			os.Exit(1)
		}

		runChat(runtime.SystemPrompt.Raw())
	},
}

func Execute(rt *Runtime) {
	runtime = rt

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
