package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"goshi/internal/config"
	"goshi/internal/detect"
	"goshi/internal/diagnose"
)

func newDoctorCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check environment health",
		RunE: func(cmd *cobra.Command, args []string) error {

			// --- detect ---
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

			// --- diagnose ---
			dg := &diagnose.BasicDiagnoser{}
			diag, err := dg.Diagnose(res)
			if err != nil {
				return err
			}

			// --- JSON output ---
			if cfg.JSON {
				out, err := json.MarshalIndent(diag, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(out))
				return nil
			}

			// --- human output ---
			if len(diag.Issues) == 0 {
				fmt.Println("✔ environment looks healthy")
				return nil
			}

			fmt.Println("Detected issues:")
			for _, issue := range diag.Issues {
				fmt.Printf(
					" - [%s][%s] %s (suggested: %s)\n",
					issue.Severity,
					issue.Code,
					issue.Message,
					issue.Strategy,
				)
			}

			// --- exit code mapping ---
			overall := diagnose.AggregateSeverity(diag.Issues)

			switch overall {
			case diagnose.SeverityOK:
				return nil
			case diagnose.SeverityWarn:
				os.Exit(1)
			case diagnose.SeverityError:
				os.Exit(2)
			case diagnose.SeverityFatal:
				os.Exit(3)
			default:
				os.Exit(2)
			}

			return nil
		},
	}
}
