package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/cshaiku/goshi/internal/version"
)

func newVersionCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show goshi version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			info := version.Current()
			switch format {
			case "json":
				out, err := json.MarshalIndent(info, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(out))
				return nil
			case "yaml":
				data, err := yaml.Marshal(info)
				if err != nil {
					return err
				}
				fmt.Print(string(data))
				return nil
			case "", "human":
				fmt.Println(version.String())
				return nil
			default:
				return fmt.Errorf("unknown format: %s (use 'json', 'yaml', or 'human')", format)
			}
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "Output format: json, yaml, or human (default: human)")
	return cmd
}
