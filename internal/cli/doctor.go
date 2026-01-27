package cli

import (
	"fmt"

  "encoding/json"

	"github.com/spf13/cobra"

	"grokgo/internal/detect"
  "grokgo/internal/diagnose"
)

func newDoctorCmd() *cobra.Command {
  var jsonOut bool

	cmd := &cobra.Command{
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

      if jsonOut {
      	out, err := json.MarshalIndent(res, "", "  ")
      	if err != nil {
      		return err
      	}

      	fmt.Println(string(out))
      	return nil
      }

    	dg := &diagnose.BasicDiagnoser{}
    	diag, err := dg.Diagnose(res)
    	if err != nil {
    		return err
    	}

			if len(res.MissingBinaries) == 0 &&
				len(res.BrokenBinaries) == 0 &&
				len(res.Warnings) == 0 {
				fmt.Println("✔ environment looks healthy")
				return nil
			}

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

  cmd.Flags().BoolVar(
	  &jsonOut,
  	"json",
	  false,
  	"output results as JSON",
  )

  return cmd
}
