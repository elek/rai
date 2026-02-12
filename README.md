# RAI - LLM from CLI

A command-line tool for interacting with multiple LLM providers. Supports Anthropic, Google (Gemini/Vertex AI), and OpenRouter with agent tools, template-based prompts, LSP, and MCP integrations.

## Features

- **Multi-provider support** - Anthropic (Claude), Google (Gemini API & Vertex AI), OpenRouter
- **Agent tools** - Built-in tools for git, file reading/listing/creation, and file editing
- **Template system** - Custom prompt templates with Go templates or Pongo2 (Django-style)
- **LSP integration** - Language Server Protocol support for code understanding (e.g., gopls)
- **MCP integration** - Model Context Protocol for connecting external tools
- **Streaming output** - Token-by-token response streaming with usage/cost reporting
- **Interactive mode** - Terminal-based REPL conversation (work in progress)

## Installation

```bash
go install github.com/elek/rai@latest
```

Or build from source:

```bash
git clone https://github.com/elek/rai.git
cd rai
go build
```

## Configuration

Create `~/.config/rai/config.yaml`:

```yaml
providers:
  - name: anthropic
    type: anthropic
    key: sk-ant-xxxxx
  - name: google
    type: google
    key: AIzaSyxxxxx
  - name: openrouter
    type: openrouter
    key: sk-or-xxxxx

models:
  - name: claude
    provider: anthropic
    model: claude-3-5-sonnet-20241022
    max_token: 2000
    temperature: 0.7
    default: true
  - name: gemini
    provider: google
    model: gemini-2.0-flash
    max_token: 4000
```

## Usage

### Ask a question

```bash
rai ask "What is the capital of France?"
rai ask --model gemini "Explain Go interfaces"
rai ask --with-tools "List all Go files in the current directory"
```

The `--with-tools` flag enables agent tools (git, file operations) so the model can interact with your local environment.

You can also specify a model inline with `provider/model` syntax:

```bash
rai ask --model anthropic/claude-3-5-sonnet-20241022 "Hello"
```

### Run custom prompts

```bash
rai do summarize myfile.txt
```

This loads a template from `~/.config/rai/summarize` and renders it with the provided arguments. Templates support XML-based prompt structure:

```xml
<model>claude</model>
<system>You are a helpful assistant.</system>
<tool name="cat"/>
<tool name="files"/>

Summarize the following file: {{index .Args 0}}
```

Template files can use either Go templates (default) or Pongo2 (prefix with `%pongo2`).

#### Template XML elements

| Element | Description |
|---------|-------------|
| `<system>` | System prompt |
| `<model>` | Model name or provider/model |
| `<tool name="...">` | Enable a built-in agent tool (git, cat, files, create, insert) |
| `<mcp command="...">` | Start an MCP server and load its tools |
| `<lsp command="...">` | Start an LSP server (e.g., `gopls`) |
| `<exec command="...">` | Execute a shell command, inline output |
| `<shell>...</shell>` | Execute a shell script block, inline output |

### List available models

```bash
rai models
rai models anthropic
```

### Interactive mode

```bash
rai run
```

Starts an interactive REPL conversation (work in progress).

## Agent Tools

When using `--with-tools` or `<tool>` elements in templates, the following tools are available:

| Tool | Description |
|------|-------------|
| `git` | Execute git commands |
| `cat` | Read file contents with optional offset/limit and line numbers |
| `files` | List files with recursive traversal and glob patterns |
| `create` | Create new files |
| `insert` | Insert content at a specific line in a file |

Additionally, LSP and MCP integrations allow extending the tool set:

- **LSP**: `<lsp command="gopls"/>` adds a `list-symbols` tool for code navigation
- **MCP**: `<mcp command="some-mcp-server"/>` loads all tools exposed by the MCP server

## Development

```bash
go build                # Build
go test ./...           # Run all tests
go vet ./...            # Lint
go fmt ./...            # Format
```
