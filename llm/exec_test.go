package llm

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutorNotifiesWhenModelReturnsNoText(t *testing.T) {
	// The model calls a tool, then ends its turn with no text at all — the
	// degenerate case where the user would otherwise see only tool calls.
	model := &scriptedModel{turns: []*Turn{
		{Blocks: []Block{{Type: BlockToolUse, ToolCallID: "c", ToolName: "noop", Input: "{}"}}, StopReason: StopToolUse},
		{Blocks: nil, StopReason: StopEnd},
	}}
	type noopIn struct{}
	noop := NewTool[noopIn]("noop", "noop", func(_ context.Context, _ noopIn) (string, error) { return "ok", nil })

	var buf bytes.Buffer
	e := &Executor{out: &buf}
	text, err := e.runAgent(context.Background(), model, "", "go", []Tool{noop})
	require.NoError(t, err)
	assert.Empty(t, text)
	assert.Contains(t, buf.String(), "no text")
}

func TestExecutorDoesNotNotifyWhenModelReturnsText(t *testing.T) {
	model := &scriptedModel{turns: []*Turn{
		{Blocks: []Block{TextBlock("here is your answer")}, StopReason: StopEnd},
	}}
	var buf bytes.Buffer
	e := &Executor{out: &buf}
	text, err := e.runAgent(context.Background(), model, "", "go", nil)
	require.NoError(t, err)
	assert.Equal(t, "here is your answer", text)
	assert.NotContains(t, buf.String(), "no text")
}
