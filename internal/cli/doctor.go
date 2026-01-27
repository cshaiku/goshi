package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"grokgo/internal/config"
	"grokgo/internal/detect"
	"grokgo/internal/diagnose"
)

func newDoctorCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check environment health",
		RunE: func(cmd *cobra.Command, args []string) error {
			d := &detect.BasicDetector{
				Binaries: []string{
					"git",
					"curl",
					"jq",
				},
			}

			res, err := d.Detect()
			if err != nil {
				return err
			}

			dg := &diagnose.BasicDiagnoser{}
			diag, err := dg.Diagnose(res)
			if err != nil {
				return err
			}

			// JSON output (global flag)
			if cfg.JSON {
				out, err := json.MarshalIndent(diag, "", "  ")
				if err != nil {
					return err
				}

				fmt.Println(string(out))
				return nil
			}

			// Human-readable output
			if len(diag.Issues) == 0 {
				fmt.Println("✔ environment looks healthy")
				return nil
			}

			fmt.Println("Detected issues:")
			for _, i := range diag.Issues {
				fmt.Printf(" - [%s] %s (suggested: %s)\n", i.Code, i.Message, i.Strategy)
			}

			return nil
		},
	}
}
