package llm

import "context"

// StopReason explains why a model turn ended.
type StopReason string

const (
	// StopEnd means the model finished its response with no further action.
	StopEnd StopReason = "end"
	// StopToolUse means the model is requesting one or more tool calls.
	StopToolUse StopReason = "tool_use"
)

// Request is a single model invocation: the system prompt, the conversation so
// far, the available tools, and generation limits.
type Request struct {
	System      string
	Messages    []Message
	Tools       []Tool
	MaxTokens   int64
	Temperature float64
}

// Turn is the assistant's response to a Request: its content blocks (text and
// any tool_use requests), token usage, and why it stopped.
type Turn struct {
	Blocks     []Block
	Usage      Usage
	StopReason StopReason
}

// Model is a provider-neutral language model. Stream performs exactly one
// request/response turn. Streamed text is delivered through onText (which may be
// nil); the complete turn — including any tool_use blocks, usage, and stop
// reason — is returned. The multi-turn tool-calling loop lives in Agent.
type Model interface {
	Provider() string
	Name() string
	Stream(ctx context.Context, req Request, onText func(delta string)) (*Turn, error)
}
