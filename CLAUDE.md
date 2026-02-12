# RAI - Go Project Guidelines

## Build Commands
```
go build                # Build the project
go run .                # Run the application
go test ./...           # Run all tests
go test ./pkg/...       # Run tests in specific package
go test -v ./pkg/... -run TestName # Run specific test
go vet ./...            # Run Go's built-in linter
```

## Code Style Guidelines
- Follow Go's official style guide: gofmt
- Run `go fmt ./...` before committing
- Use `golangci-lint run` for comprehensive linting
- Use descriptive camelCase for variable names, PascalCase for exported names
- Group imports: standard library, third-party, internal packages
- Error handling: always check errors, use meaningful error messages. Use `github.com/pkg/errors`
- Prefer explicit error returns over panics
- Use context.Context for request-scoped values/cancellation
- Document all exported functions, types, and constants
- Write tests for all exported functions. Use `github.com/stretchr/testify/assert`

## Project Architecture

### Package Structure
- `main.go` - Entry point, Kong CLI setup with 4 subcommands (ask, do, run, models)
- `cmd/` - CLI command implementations (ask, do, models, usage reporting)
- `config/` - YAML config loading from `~/.config/rai/config.yaml`, model/provider definitions
- `llm/` - LLM model creation (Anthropic, Google, OpenRouter) and prompt execution with streaming
- `console/` - Interactive REPL using Bubbletea TUI framework (partially implemented)
- `tool/` - Agent tools (git, cat, files, create, insert) + LSP and MCP integrations
- `templates/` - Dual template engine (Go templates, Pongo2) with XML-based prompt structure

### Key Libraries
- `charm.land/fantasy` - Unified LLM interface with streaming and agent tools
- `github.com/alecthomas/kong` - CLI framework
- `github.com/charmbracelet/bubbletea` - Terminal UI
- `github.com/elek/lspc` - LSP client for code understanding
- `github.com/modelcontextprotocol/go-sdk` - MCP SDK for external tool integration
- `github.com/flosch/pongo2` - Django-style template engine

### Configuration
Config lives at `~/.config/rai/config.yaml` with `providers` (name, type, key) and `models` (name, provider, model, max_token, temperature, default) sections.

### Template System
Custom prompt templates stored in `~/.config/rai/{command}` files. Templates use XML elements:
- `<system>` - System prompt
- `<model>` - Model selection
- `<tool name="...">` - Enable agent tools
- `<mcp command="...">` / `<lsp command="...">` - External tool servers
- `<exec command="...">` / `<shell>` - Inline shell execution
