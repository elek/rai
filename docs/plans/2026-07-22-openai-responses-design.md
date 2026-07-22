# OpenAI Responses API provider

Date: 2026-07-22

## Problem

The default OpenAI reasoning model (`gpt-5.5`) intermittently degenerates in the
multi-turn tool loop: it leaks reasoning tokens into tool-call arguments,
produces malformed tool calls, and returns no final text. The user sees the tool
calls but no answer.

Investigation established:

- rai's streaming is correct: the streamed text equals the accumulated final
  text on every turn.
- Non-reasoning OpenAI models (`gpt-4o-mini`, `gpt-4.1-mini`) run the same
  `review` scenario reliably (4/4), on the same Chat Completions code path.
- Non-reasoning third-party models (Kimi via together) also work.
- Only reasoning models degenerate, and only in the multi-turn tool loop.

Root cause: reasoning models are designed for the **Responses API**, which
preserves the model's reasoning across tool-call turns via reasoning items. The
Chat Completions API discards reasoning between turns, so a reasoning model
re-derives its plan from scratch on every tool round-trip.

## Approach

Add a second OpenAI provider implementation backed by the Responses API, keep
the existing Chat Completions implementation unchanged, and let the provider
`type` in config select between them. Reasoning is round-tripped statelessly
(Option A): the encrypted reasoning item travels through the neutral model as a
new block type, so the existing agent loop carries it back for free.

## Config selection

`llm/provider.go` gains one switch case:

```go
case "openai-responses":
    return NewOpenAIResponsesModel(p.Key, p.Endpoint, model.Model, maxTokens, model.Debug), nil
```

No new config fields — `openai-responses` reuses `Key` and `Endpoint`.

```yaml
providers:
  - name: openai-reasoning
    type: openai-responses
    key: sk-...
models:
  - name: gpt55
    provider: openai-reasoning
    model: gpt-5.5
```

`type: openai-responses` targets real OpenAI (or Azure OpenAI). OpenAI-compatible
endpoints (together, chutes, openrouter) generally do not implement
`/v1/responses`; those keep `type: openai`. A 404 there means "use `type: openai`".

## Neutral-model change

One addition to `llm/types.go`:

```go
// BlockReasoning carries a reasoning item produced by a reasoning model. Its
// content is opaque (provider-encrypted) and exists only to be sent back on
// later turns so the model retains its chain of thought across tool calls.
// Providers that don't understand it ignore it.
BlockReasoning BlockType = "reasoning"
```

It reuses existing `Block` fields, so `Block` is unchanged:

- `Block.ToolCallID` -> the reasoning item's `id`
- `Block.Text` -> the opaque `encrypted_content`

`BlockReasoning` is never streamed to `onText` and never rendered as an answer
(`textOf()` only concatenates `BlockText`). It is invisible conversation state
that happens to travel inside `Turn.Blocks`.

Other providers already switch on block type with no error default
(`anthropic.go`, `openai.go`), so they silently skip `BlockReasoning`.

## Round-trip flow

No agent-loop changes.

1. Turn 1 response: the Responses model streams text via `onText` and builds
   `Turn.Blocks` in stream order: `[BlockReasoning, BlockToolUse, ...]` —
   reasoning first, then the tool calls it produced.
2. Agent loop appends `Message{Role: RoleAssistant, Blocks: turn.Blocks}`, runs
   the tools, appends the `RoleTool` result message. Unchanged.
3. Turn 2 request: the Responses model walks `Request.Messages` and rebuilds the
   API `Input` list, preserving order so each reasoning item precedes its
   function calls:
   - user text -> message item
   - `BlockReasoning` -> reasoning input item (id + encrypted_content)
   - `BlockToolUse` -> function_call item (call_id, name, arguments)
   - `BlockToolResult` -> function_call_output item (call_id, output)

## Request/response mapping (`llm/openai_responses.go`)

Request:

```go
params := responses.ResponseNewParams{
    Model:   m.model,
    Input:   toResponsesInput(req.System, req.Messages),
    Store:   openai.Bool(false),
    Include: []responses.ResponseIncludable{
        responses.ResponseIncludableReasoningEncryptedContent,
    },
}
if len(req.Tools) > 0 { params.Tools = toResponsesTools(req.Tools) }
if m.maxTokens > 0 { params.MaxOutputTokens = openai.Int(m.maxTokens) }
```

`store: false` paired with the `reasoning.encrypted_content` include is
load-bearing: without the include, `store:false` would strip reasoning between
turns. This pairing is commented in the code.

Stream consumption (`client.Responses.NewStreaming`), dispatching on
`event.Type`:

- `response.output_text.delta` -> accumulate text and call `onText(delta)`.
- `response.output_item.done` -> inspect the item:
  - reasoning item -> `BlockReasoning{ToolCallID: id, Text: encrypted_content}`
  - function_call item -> `BlockToolUse{ToolCallID: call_id, ToolName: name, Input: arguments}`, set `StopReason = StopToolUse`
- `response.completed` -> read `Response.Usage` for token counts.
- `error` -> return the error.

Turn assembly: blocks in stream order, text block first if any text
accumulated; default `StopReason = StopEnd`, upgraded to `StopToolUse` when any
function call appears (mirrors `turnFromOpenAI`).

Detail to verify during implementation: the reasoning input-item param marks
`summary` as required; pass an empty summary slice with the encrypted content
and confirm it round-trips against the live API.

## Testing

Unit tests (no network), the bulk of coverage:

- `toResponsesInput` round-trip: `[user text] -> [assistant: reasoning + 2
  tool_use] -> [tool results]` produces the right item types in the right order
  with `call_id`/`name`/`arguments`/`encrypted_content` in the right fields.
- `toResponsesTools`: neutral tools -> responses function-tool params.
- Turn assembly helper: text-only -> `StopEnd`; text + function calls ->
  `StopToolUse` with reasoning/tool blocks in order.

Integration test (gated on API key env var, following the existing
`openai_integration_test.go` / `anthropic_integration_test.go` pattern):
`openai_responses_integration_test.go` runs a small multi-turn tool scenario
against a real reasoning model and asserts a non-empty final answer. This proves
the encrypted-content round-trip against the live API.

Manual verification: re-run the `review` scenario with a reasoning model several
times and confirm it no longer degenerates.

## Error handling & edge cases

- Turn with no reasoning: nothing to capture, no `BlockReasoning`; works.
- Empty final text: already handled by the `exec.go` empty-response notice.
- Stream/API error: returned as a Go error, propagates to `main`.
- Backward compatibility: no change to `type: openai`; existing configs work.
- Context cancellation: the stream loop honors `ctx`/`stream.Err()`.

## Implementation notes (from live-API validation)

Two things the real Responses API enforced, both now covered by tests:

- **Strict tool schema.** Unlike Chat Completions, the Responses API rejects a
  function schema whose `required` is `null` or whose `properties` is `null`
  (`"None is not of type 'array'"`). `toResponsesTools` normalizes a param-less
  tool's nil slice/map to `[]`/`{}`.
- **Assistant text in tool-call turns.** When the model emits visible text
  alongside its reasoning and tool calls, that `BlockText` is mapped to an
  assistant-role message so it is preserved (not dropped) in the round-trip.

Validated end to end: `gpt-5.5` over the Responses API completes the multi-turn
`review` scenario coherently, where the same model on Chat Completions
intermittently degenerated.

## Scope (YAGNI)

Deferred: a `reasoning_effort` config knob, streaming/displaying reasoning
summaries, built-in tools (web/file search), image/audio content.
