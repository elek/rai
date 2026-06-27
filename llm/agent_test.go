package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptedModel returns a predefined sequence of turns, one per Stream call.
type scriptedModel struct {
	turns []*Turn
	calls int
	// lastRequest captures the Request passed to the most recent Stream call.
	lastRequest Request
}

func (m *scriptedModel) Provider() string { return "scripted" }
func (m *scriptedModel) Name() string     { return "scripted" }

func (m *scriptedModel) Stream(_ context.Context, req Request, onText func(string)) (*Turn, error) {
	m.lastRequest = req
	turn := m.turns[m.calls]
	m.calls++
	if onText != nil {
		for _, b := range turn.Blocks {
			if b.Type == BlockText {
				onText(b.Text)
			}
		}
	}
	return turn, nil
}

func TestAgentReturnsTextWhenNoToolCalls(t *testing.T) {
	model := &scriptedModel{turns: []*Turn{
		{Blocks: []Block{TextBlock("Madrid is the capital.")}, StopReason: StopEnd, Usage: Usage{InputTokens: 5, OutputTokens: 4, TotalTokens: 9}},
	}}

	agent := NewAgent(model, "", nil)
	var streamed string
	res, err := agent.Run(context.Background(), "Capital of Spain?", RunOptions{
		OnTextDelta: func(d string) { streamed += d },
	})
	require.NoError(t, err)
	assert.Equal(t, "Madrid is the capital.", res.Text)
	assert.Equal(t, "Madrid is the capital.", streamed)
	assert.Equal(t, int64(9), res.Usage.TotalTokens)
}

func TestAgentExecutesToolThenReturnsFinalText(t *testing.T) {
	model := &scriptedModel{turns: []*Turn{
		// Turn 1: ask to call the tool.
		{
			Blocks:     []Block{{Type: BlockToolUse, ToolCallID: "call-1", ToolName: "echo", Input: `{"text":"hi"}`}},
			StopReason: StopToolUse,
			Usage:      Usage{InputTokens: 10, OutputTokens: 2, TotalTokens: 12},
		},
		// Turn 2: final answer.
		{
			Blocks:     []Block{TextBlock("done: hi")},
			StopReason: StopEnd,
			Usage:      Usage{InputTokens: 8, OutputTokens: 3, TotalTokens: 11},
		},
	}}

	type echoIn struct {
		Text string `json:"text"`
	}
	var ran bool
	echo := NewTool[echoIn]("echo", "echoes", func(_ context.Context, in echoIn) (string, error) {
		ran = true
		return "echoed " + in.Text, nil
	})

	agent := NewAgent(model, "sys", []Tool{echo})
	var toolCalls []string
	res, err := agent.Run(context.Background(), "go", RunOptions{
		OnToolCall: func(name, input string) { toolCalls = append(toolCalls, name) },
	})
	require.NoError(t, err)

	assert.True(t, ran, "tool should have been executed")
	assert.Equal(t, []string{"echo"}, toolCalls)
	assert.Equal(t, "done: hi", res.Text)
	// Usage accumulates across both turns.
	assert.Equal(t, int64(23), res.Usage.TotalTokens)

	// The second request must include: user prompt, assistant tool_use, tool result.
	require.Len(t, model.lastRequest.Messages, 3)
	assert.Equal(t, RoleUser, model.lastRequest.Messages[0].Role)
	assert.Equal(t, RoleAssistant, model.lastRequest.Messages[1].Role)
	assert.Equal(t, RoleTool, model.lastRequest.Messages[2].Role)
	resultBlock := model.lastRequest.Messages[2].Blocks[0]
	assert.Equal(t, BlockToolResult, resultBlock.Type)
	assert.Equal(t, "call-1", resultBlock.ToolCallID)
	assert.Equal(t, "echoed hi", resultBlock.Text)
}

func TestAgentStopsAtMaxSteps(t *testing.T) {
	// A model that always asks for a tool would loop forever without MaxSteps.
	loopTurn := &Turn{
		Blocks:     []Block{{Type: BlockToolUse, ToolCallID: "c", ToolName: "noop", Input: `{}`}},
		StopReason: StopToolUse,
	}
	model := &scriptedModel{turns: []*Turn{loopTurn, loopTurn, loopTurn, loopTurn, loopTurn}}

	type noopIn struct{}
	noop := NewTool[noopIn]("noop", "noop", func(_ context.Context, _ noopIn) (string, error) { return "ok", nil })

	agent := NewAgent(model, "", []Tool{noop})
	_, err := agent.Run(context.Background(), "go", RunOptions{MaxSteps: 3})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max steps")
	assert.Equal(t, 3, model.calls)
}
