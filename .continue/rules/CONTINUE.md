# Goshi Project Guide

## Project Overview
Goshi is a CLI application that integrates with LLM (Language Learning Models) providers, primarily focused on using Ollama. The project is written in Go and provides a robust configuration system for managing LLM interactions.

### Key Technologies
- Go (Golang)
- Ollama integration
- YAML configuration
- CLI interface

## Project Structure
```
goshi/
├── .continue/           # Continue extension configuration
├── .goshi/             # Goshi-specific configuration
├── internal/           # Internal packages
│   ├── actions/        # Action implementations
│   ├── app/           # Core application logic
│   ├── cli/           # CLI interface components
│   ├── config/        # Configuration management
│   ├── llm/           # LLM integration
│   ├── selfmodel/     # Self-model management
│   └── ...            # Other internal packages
├── bin/               # Binary outputs
└── main.go           # Application entry point
```

## Key Components

### Configuration System
- Located in `internal/config/`
- Supports multiple configuration sources:
  1. Environment variables (GOSHI_*)
  2. Repository-scoped config (.goshi.yaml)
  3. User home config (~/.goshi/config.yaml)
  4. System-wide config (/etc/goshi/config.yaml)
- Key configuration types:
  - LLMConfig: Model settings (provider, temperature, tokens, etc.)
  - SafetyConfig: Safety features (dry-run, permissions, backups)
  - LoggingConfig: Logging preferences
  - BehaviorConfig: Runtime behavior settings

### LLM Integration
- Primarily configured for Ollama integration
- Default model: mistral
- Configurable parameters:
  - Temperature
  - Max tokens
  - Request timeout
  - Local server URL/port

### CLI Runtime
- Manages system prompts and execution flow
- Handles command processing and user interactions

## Development Workflow

### Setting Up Development Environment
1. Ensure Go is installed
2. Clone the repository
3. Install dependencies: `go mod download`
4. Build: `go build`

### Key Files
- `main.go`: Application entry point
- `goshi.yaml`: Main configuration file
- `goshi.self.model.yaml`: Self-model configuration
- `goshi.threat.model.md`: Security threat model

### Configuration
Default configuration can be overridden through:
- Environment variables:
  - GOSHI_MODEL
  - GOSHI_LLM_PROVIDER
  - GOSHI_OLLAMA_URL
  - GOSHI_OLLAMA_PORT
- Configuration files:
  - .goshi.yaml
  - ~/.goshi/config.yaml

## Common Tasks

### Running the Application
```bash
# Build and run
go build
./goshi [commands]

# Development run
go run main.go [commands]
```

### Configuration Validation
The config package includes validation for:
- LLM model and provider settings
- Temperature range (0-2)
- Token limits
- Port ranges
- Logging levels

### Testing
Run tests with:
```bash
go test ./...
```

## Troubleshooting

### Common Issues
1. Configuration loading failures
   - Check file permissions
   - Verify YAML syntax
   - Ensure correct file locations

2. LLM Connection Issues
   - Verify Ollama is running
   - Check port configuration
   - Validate URL settings

### Debugging Tips
- Set logging level to "debug" in configuration
- Use environment variables for quick configuration changes
- Check ~/.goshi/cache/ for cached data

## References
- [Project Changelog](CHANGELOG.md)
- [Migration Guide](MIGRATION.md)
- [LLM Integration](LLM_INTEGRATION.md)
- [Threat Model](goshi.threat.model.md)

---
*Note: This documentation is maintained by the Continue extension and may be updated automatically based on project changes.*