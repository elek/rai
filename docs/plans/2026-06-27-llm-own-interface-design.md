# LLM package refactor: own interface, drop `charm.land/fantasy`

Date: 2026-06-27

## Goal

Replace the `charm.land/fantasy` dependency with an in-repo LLM abstraction and
provider implementations built on the official Go SDKs. Support **openai**,
**anthropic**, and **fake** providers in this phase. Remove **google** and
**openrouter** (re-addable later against the new interface).

## What fantasy gave us (and we now re-implement)

1. `LanguageModel` — the model interface.
2. The agent loop (`NewAgent` + `Stream`) — multi-turn tool calling.
3. The tool abstraction (`AgentTool`, `ToolResponse`, `NewAgentTool[T]` with
   reflection-based JSON schema from struct tags).
4. `Usage`.

## Package layout

All new types live in package `llm`. The `tool` package imports `llm` for the
tool interface. No import cycle: `llm` never imports `tool`. `templates`, `acp`,
and `cmd` import both.

## Core data types (`llm/types.go`)

```go
type Usage struct{ InputTokens, OutputTokens, TotalTokens int64 }

type Role string // "user" | "assistant" | "tool"

type BlockType string // "text" | "tool_use" | "tool_result"

type Block struct {
    Type       BlockType
    Text       string // text content, or tool_result payload
    ToolCallID string // tool_use: the call id; tool_result: which call
    ToolName   string // tool_use
    Input      string // tool_use: raw JSON args
    IsError    bool   // tool_result
}

type Message struct { Role Role; Blocks []Block }
```

## Model interface (`llm/model.go`)

One streamed turn, provider-neutral.

```go
type Request struct {
    System      string
    Messages    []Message
    Tools       []Tool
    MaxTokens   int64
    Temperature float64
}

type StopReason string // "end" | "tool_use"

type Turn struct { Blocks []Block; Usage Usage; StopReason StopReason }

type Model interface {
    Provider() string
    Name() string
    Stream(ctx context.Context, req Request, onText func(delta string)) (*Turn, error)
}
```

## Tool abstraction (`llm/tool.go`)

```go
type ToolInfo struct {
    Name, Description string
    Parameters map[string]any // JSON-schema properties
    Required   []string
}
type ToolCall   struct { ID, Name, Input string }
type ToolResult struct { Content string; IsError bool }

type Tool interface {
    Info() ToolInfo
    Run(ctx context.Context, call ToolCall) (ToolResult, error)
}

func NewTool[T any](name, desc string, fn func(ctx context.Context, in T) (string, error)) Tool
```

`NewTool[T]` reflects over `T`'s `json`/`description` struct tags to build the
schema, unmarshals `Input` into `T`, calls `fn`. Replaces
`fantasy.NewAgentTool[T]`. MCP/LSP tools implement `Tool` directly (raw schema).

## Agent loop (`llm/agent.go`)

```go
type Agent struct { model Model; tools []Tool; system string }
func NewAgent(model Model, system string, tools []Tool) *Agent

type RunOptions struct {
    OnTextDelta func(delta string)
    OnToolCall  func(name, input string)
    MaxSteps    int // default 50
}
type Result struct { Text string; Usage Usage }

func (a *Agent) Run(ctx context.Context, prompt string, opts RunOptions) (*Result, error)
```

Loop: seed `messages=[user(prompt)]`; each step call `model.Stream` (forward
deltas); append assistant turn; if `StopReason != tool_use` return accumulated
text + summed usage; else run each `tool_use` block (fire `OnToolCall`), append
a `tool` message of `tool_result` blocks; repeat up to `MaxSteps`.

## Provider mapping

- **Anthropic** (`github.com/anthropics/anthropic-sdk-go`): `Block`↔content
  blocks, tools→`ToolUnionParam`, `StreamingNew`+`Accumulate`,
  `StopReasonToolUse`→`tool_use`.
- **OpenAI** (`github.com/openai/openai-go`): `Message`→chat messages (assistant
  `tool_calls`, `role:"tool"` results), `Tool`→function tool, streamed
  `tool_calls` accumulation.
- **Fake**: random text word-by-word, never emits tool calls.

## Call-site changes

- `llm/exec.go`: use `Agent.Run`.
- `acp/server.go`: use `Agent.Run`, map deltas/tool calls to notifications.
- `templates.ParsedTemplate.Tools`: `[]llm.Tool`.
- `tool/registry.go`, `tool/mcp.go`, `tool/lsp.go`: `fantasy.*`→`llm.*`.
- `cmd/usage.go`: `llm.Usage` + `Model.Provider()/Name()`.
- Drop `charm.land/fantasy` from go.mod.

## Testing (TDD)

- `NewTool` schema reflection + Run unmarshalling.
- Agent loop with a fake tool-emitting model: tool_use → result → final text,
  usage summed across turns, MaxSteps guard.
- Fake model streaming (deltas reassemble to full text).
- Provider block-translation helpers as pure functions.
- **Integration tests** (`anthropic_integration_test.go`,
  `openai_integration_test.go`): if the provider API key env var is set, hit the
  real endpoint with "What is the capital of Spain?" and assert the response
  contains "madrid" (case-insensitive); otherwise `t.Skip`.
```
