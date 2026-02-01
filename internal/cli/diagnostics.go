package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"goshi/internal/config"
	"goshi/internal/selfmodel"
)

type diagnosticsReport struct {
	Timestamp string   `json:"timestamp"`
	Status    string   `json:"status"`
	Findings  []string `json:"findings"`
}

func newDiagnosticsCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diagnostics",
		Short: "Validate Goshi against its self-model",
		RunE: func(cmd *cobra.Command, args []string) error {
			model, err := selfmodel.Load("")
			if err != nil {
				return err // internal error
			}

			var findings []string

			// --- required binaries ---
			for _, bin := range model.Deps.RequiredBinaries {
				if _, err := exec.LookPath(bin.Name); err != nil {
					findings = append(findings, "Missing binary: "+bin.Name)
				}
			}

			// --- required directories ---
			for _, dir := range model.Structure.RequiredDirectories {
				info, err := os.Stat(dir)
				if err != nil || !info.IsDir() {
					findings = append(findings, "Missing directory: "+dir)
				}
			}

			ts := time.Now().Format("2006-01-02 15:04:05")
			isJSON, _ := cmd.Flags().GetBool("json")

			report := diagnosticsReport{
				Timestamp: ts,
				Findings:  findings,
			}

			if len(findings) > 0 {
				report.Status = "failure"

				if isJSON {
					_ = json.NewEncoder(os.Stdout).Encode(report)
				} else {
					fmt.Printf("%s Diagnostic Failure\n", ts)
					for _, f := range findings {
						fmt.Printf("* %s\n", f)
					}
				}

				os.Exit(2)
			}

			report.Status = "success"

			if isJSON {
				_ = json.NewEncoder(os.Stdout).Encode(report)
			} else {
				fmt.Printf("%s Diagnostic Success\n", ts)
			}

			return nil
		},
	}

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	return cmd
}
