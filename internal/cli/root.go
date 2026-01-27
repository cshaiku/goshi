package cli

import (
	"log"

	"github.com/spf13/cobra"

	"grokgo/internal/app"
	"grokgo/internal/config"
)


func NewRootCmd() *cobra.Command {
	cfg := config.Load()

	var modelFlag string
	var dryRunFlag bool
  var yesFlag bool
  var jsonFlag bool

	cmd := &cobra.Command{
		Use:   "grokgo",
		Short: "GrokGo — Go-native AI CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if modelFlag != "" {
				cfg.Model = modelFlag
			}

			if cmd.Flag("dry-run").Changed {
				cfg.DryRun = dryRunFlag
			}

      if cmd.Flag("yes").Changed {
        cfg.Yes = yesFlag
      }

      if cmd.Flag("json").Changed {
      	cfg.JSON = jsonFlag
      }

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			app.Run(cfg)
		},
	}

	cmd.PersistentFlags().BoolVar(
		&dryRunFlag,
		"dry-run",
		true,
		"do not execute changes, only show what would happen",
	)

  cmd.PersistentFlags().BoolVar(
  	&yesFlag,
  	"yes",
  	false,
  	"automatically confirm execution",
  )

  cmd.PersistentFlags().BoolVar(
  	&jsonFlag,
  	"json",
  	false,
  	"output machine-readable JSON",
  )

	cmd.Flags().StringVar(
		&modelFlag,
		"model",
		"",
		"override model (env: GROKGO_MODEL)",
	)

	cmd.AddCommand(newDoctorCmd(&cfg))
  cmd.AddCommand(newHealCmd(&cfg))

	return cmd
}


func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}
