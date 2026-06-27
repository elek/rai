package llm

import (
	"context"

	"github.com/pkg/errors"
)

// defaultMaxSteps bounds the number of model turns in a single Run, guarding
// against a model that keeps requesting tools indefinitely.
const defaultMaxSteps = 50

// Agent drives a multi-turn tool-calling loop on top of a Model.
type Agent struct {
	model  Model
	system string
	tools  []Tool
}

// NewAgent creates an Agent bound to a model, system prompt, and tool set.
func NewAgent(model Model, system string, tools []Tool) *Agent {
	return &Agent{model: model, system: system, tools: tools}
}

// RunOptions configures a single Run.
type RunOptions struct {
	// OnTextDelta is called for each streamed chunk of assistant text.
	OnTextDelta func(delta string)
	// OnToolCall is called when the assistant requests a tool, with the tool
	// name and its raw JSON input.
	OnToolCall func(name, input string)
	// MaxSteps bounds the number of model turns. Zero uses defaultMaxSteps.
	MaxSteps int
}

// Result is the outcome of an agent Run.
type Result struct {
	Text  string
	Usage Usage
}

// Run sends prompt to the model and loops: each turn, it streams the assistant
// response, and if the model requested tools, executes them and feeds the
// results back. It returns once the model stops requesting tools, or errors if
// MaxSteps is exceeded.
func (a *Agent) Run(ctx context.Context, prompt string, opts RunOptions) (*Result, error) {
	maxSteps := opts.MaxSteps
	if maxSteps <= 0 {
		maxSteps = defaultMaxSteps
	}

	byName := make(map[string]Tool, len(a.tools))
	for _, t := range a.tools {
		byName[t.Info().Name] = t
	}

	messages := []Message{UserMessage(prompt)}
	var (
		usage    Usage
		lastText string
	)

	for step := 0; step < maxSteps; step++ {
		turn, err := a.model.Stream(ctx, Request{
			System:   a.system,
			Messages: messages,
			Tools:    a.tools,
		}, opts.OnTextDelta)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		usage = usage.Add(turn.Usage)

		// Record the assistant turn and capture its text.
		messages = append(messages, Message{Role: RoleAssistant, Blocks: turn.Blocks})
		lastText = textOf(turn.Blocks)

		toolUses := toolUseBlocks(turn.Blocks)
		if turn.StopReason != StopToolUse && len(toolUses) == 0 {
			return &Result{Text: lastText, Usage: usage}, nil
		}

		// Execute each requested tool and collect the results into one tool turn.
		results := make([]Block, 0, len(toolUses))
		for _, tu := range toolUses {
			if opts.OnToolCall != nil {
				opts.OnToolCall(tu.ToolName, tu.Input)
			}
			results = append(results, a.runTool(ctx, byName, tu))
		}
		messages = append(messages, Message{Role: RoleTool, Blocks: results})
	}

	return nil, errors.Errorf("agent exceeded max steps (%d) without completing", maxSteps)
}

// runTool invokes the named tool and returns a tool_result block.
func (a *Agent) runTool(ctx context.Context, byName map[string]Tool, tu Block) Block {
	tool, ok := byName[tu.ToolName]
	if !ok {
		return Block{Type: BlockToolResult, ToolCallID: tu.ToolCallID, Text: "unknown tool: " + tu.ToolName, IsError: true}
	}
	res, err := tool.Run(ctx, ToolCall{ID: tu.ToolCallID, Name: tu.ToolName, Input: tu.Input})
	if err != nil {
		return Block{Type: BlockToolResult, ToolCallID: tu.ToolCallID, Text: err.Error(), IsError: true}
	}
	return Block{Type: BlockToolResult, ToolCallID: tu.ToolCallID, Text: res.Content, IsError: res.IsError}
}

// toolUseBlocks returns the tool_use blocks from a slice of blocks.
func toolUseBlocks(blocks []Block) []Block {
	var out []Block
	for _, b := range blocks {
		if b.Type == BlockToolUse {
			out = append(out, b)
		}
	}
	return out
}

// textOf concatenates the text blocks in a slice.
func textOf(blocks []Block) string {
	var s string
	for _, b := range blocks {
		if b.Type == BlockText {
			s += b.Text
		}
	}
	return s
}
