package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cshaiku/goshi/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage goshi configuration",
		Long: `Manage and inspect goshi configuration.

Config file search order:
  1. $GOSHI_CONFIG environment variable
  2. .goshi.yaml (current directory)
  3. ~/.goshi/config.yaml (home directory)
  4. /etc/goshi/config.yaml (system-wide)
  5. Built-in defaults

QUICK START:
  $ goshi config show              # Display current config
  $ goshi config show --format=yaml  # Show as YAML
  $ goshi config validate          # Validate current config
  $ goshi config init --output ~/.goshi/config.yaml  # Generate template

SEE ALSO:
  goshi help config show      - Display configuration
  goshi help config validate  - Validate configuration file
  goshi help config init      - Generate config template

ENVIRONMENT:
  GOSHI_CONFIG        - Path to configuration file (overrides file search)
  GOSHI_MODEL         - LLM model to use (overrides config file)
  GOSHI_LLM_PROVIDER  - LLM provider: ollama, openai, etc. (overrides config file)
  GOSHI_OLLAMA_URL    - Ollama server URL (overrides config file)
  GOSHI_OLLAMA_PORT   - Ollama server port number (overrides config file)`,
	}

	cmd.AddCommand(
		newConfigShowCommand(),
		newConfigValidateCommand(),
		newConfigInitCommand(),
	)

	return cmd
}

func newConfigShowCommand() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show currently loaded configuration",
		Long: `Display the currently loaded configuration.

Shows all configuration values after merging:
  1. Config file (if found)
  2. Environment variable overrides
  3. Default values for unset options

By default, outputs in JSON format for easy parsing.

EXAMPLES:
  $ goshi config show
  
  $ goshi config show --format=yaml

  $ goshi config show | jq '.llm.model'

ENVIRONMENT:
  GOSHI_MODEL         - Overrides LLM model setting
  GOSHI_LLM_PROVIDER  - Overrides LLM provider setting
  GOSHI_OLLAMA_URL    - Overrides Ollama server URL
  GOSHI_OLLAMA_PORT   - Overrides Ollama server port

EXIT CODES:
  0   - Success
  1   - Unknown format specified`,

		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig()

			switch format {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(cfg)

			case "yaml":
				data, err := yaml.Marshal(cfg)
				if err != nil {
					return fmt.Errorf("failed to marshal config: %w", err)
				}
				fmt.Print(string(data))
				return nil

			default:
				return fmt.Errorf("unknown format: %s (use 'json' or 'yaml')", format)
			}
		},
	}

	cmd.Flags().StringVar(&format, "format", "json", "Output format (json or yaml)")
	return cmd
}

func newConfigValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate [config-file]",
		Short: "Validate configuration file or currently loaded config",
		Long: `Validate configuration for correctness.

If no config file is specified, validates the currently loaded configuration.
Otherwise, loads and validates the specified config file.

Checks:
  - Temperature is between 0 and 2
  - Port is between 1 and 65535
  - MaxTokens and TimeoutSeconds are positive
  - Required fields (model, provider) are set
  - Logging level is valid (debug, info, warn, error)

EXAMPLES:
  $ goshi config validate
  ✓ Configuration is valid

  $ goshi config validate path/to/config.yaml
  ✓ Configuration is valid

  $ goshi config validate /etc/goshi/config.yaml
  Error: invalid temperature: 5 (must be 0-2)

ENVIRONMENT:
  GOSHI_CONFIG  - Path to config file to validate (used if no argument provided)

EXIT CODES:
  0   - Configuration is valid
  1   - Validation failed or file read error`,

		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg config.Config

			if len(args) > 0 {
				// Load specified config file
				path := args[0]
				data, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("failed to read config file: %w", err)
				}

				if err := yaml.Unmarshal(data, &cfg); err != nil {
					return fmt.Errorf("failed to parse config file: %w", err)
				}
			} else {
				// Validate currently loaded config
				cfg = *GetConfig()
			}

			// Apply defaults if needed
			if cfg.LLM.Model == "" {
				cfg.LLM.Model = config.LoadDefaults().LLM.Model
			}
			if cfg.LLM.Provider == "" {
				cfg.LLM.Provider = config.LoadDefaults().LLM.Provider
			}

			if err := cfg.Validate(); err != nil {
				fmt.Fprintf(os.Stderr, "✗ Configuration invalid: %v\n", err)
				return err
			}

			if len(args) > 0 {
				fmt.Printf("✓ Configuration file '%s' is valid\n", args[0])
			} else {
				fmt.Println("✓ Currently loaded configuration is valid")
			}

			return nil
		},
	}
}

func newConfigInitCommand() *cobra.Command {
	var output string
	var overwrite bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a default configuration file",
		Long: `Generate a default configuration file template.

Creates a well-commented configuration file with all available options
and sensible defaults. Can write to:
	- Current directory (.goshi.yaml)
	- Home directory (~/.goshi/config.yaml)
	- Custom path

Use --output to specify where to write the file.
Use --overwrite to replace existing files.

EXAMPLES:
	$ goshi config init
	✓ Created .goshi.yaml in current directory

	$ goshi config init --output ~/.goshi/config.yaml
	✓ Created ~/.goshi/config.yaml

	$ goshi config init --output /etc/goshi/config.yaml --overwrite
	✓ Created /etc/goshi/config.yaml

ENVIRONMENT:
  GOSHI_CONFIG  - Ignored by init; config is created in specified location

EXIT CODES:
  0   - Config file created successfully
  1   - File already exists or directory creation failed`,

		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine output path
			outPath := output
			if outPath == "" {
				outPath = ".goshi.yaml"
			} else if outPath[0] == '~' {
				// Expand home directory
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("failed to get home directory: %w", err)
				}
				outPath = filepath.Join(home, outPath[1:])
			}

			// Check if file exists and overwrite flag
			if _, err := os.Stat(outPath); err == nil && !overwrite {
				return fmt.Errorf("file already exists: %s (use --overwrite to replace)", outPath)
			}

			// Create parent directories if needed
			dir := filepath.Dir(outPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

			// Generate template
			cfg := config.LoadDefaults()
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("failed to generate config template: %w", err)
			}

			// Add header comments
			header := `# Goshi Configuration
	# Generated by 'goshi config init'
	# 
	# This file configures goshi's behavior including:
	# - LLM model and provider settings
	# - Local Ollama server connection details
	# - Safety and permission handling
	# - Logging configuration
	#
	# For more information, see: goshi help config

	`

			fullContent := header + string(data)

			// Write file
			if err := os.WriteFile(outPath, []byte(fullContent), 0644); err != nil {
				return fmt.Errorf("failed to write config file: %w", err)
			}

			fmt.Printf("✓ Created %s\n", outPath)
			fmt.Printf("  Edit this file to customize goshi configuration\n")
			fmt.Printf("  Run 'goshi config validate' to check for errors\n")

			return nil
		},
	}

	cmd.Flags().StringVar(&output, "output", "", "Output path for config file (default: .goshi.yaml)")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing file if present")
	return cmd
}
