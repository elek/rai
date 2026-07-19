package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeModelStreamsReassembleToFullText(t *testing.T) {
	m := NewFakeModel("fake", "fake-1")
	assert.Equal(t, "fake", m.Provider())
	assert.Equal(t, "fake-1", m.Name())

	var streamed strings.Builder
	turn, err := m.Stream(context.Background(), Request{}, func(d string) {
		streamed.WriteString(d)
	})
	require.NoError(t, err)
	require.Len(t, turn.Blocks, 1)

	full := turn.Blocks[0].Text
	assert.Equal(t, full, streamed.String(), "streamed deltas should reassemble to the full text")
	assert.Equal(t, StopEnd, turn.StopReason)
	assert.Positive(t, turn.Usage.OutputTokens)
}

func TestFakeModelHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewFakeModel("fake", "fake-1").Stream(ctx, Request{}, nil)
	require.Error(t, err)
}

func TestFakeModelDrivesAgent(t *testing.T) {
	agent := NewAgent(NewFakeModel("fake", "fake-1"), "", nil)
	res, err := agent.Run(context.Background(), "hello", RunOptions{})
	require.NoError(t, err)
	assert.NotEmpty(t, res.Text)
}

func TestFakeModelCommitRequestsGitTools(t *testing.T) {
	m := NewFakeModel("fake", "fake-1")
	turn, err := m.Stream(context.Background(), Request{
		Messages: []Message{UserMessage("please commit my changes")},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, StopToolUse, turn.StopReason)

	toolUses := toolUseBlocks(turn.Blocks)
	require.Len(t, toolUses, 2)
	for _, tu := range toolUses {
		assert.Equal(t, "git", tu.ToolName)
		assert.NotEmpty(t, tu.ToolCallID)
	}
	assert.Contains(t, toolUses[0].Input, "git add -A")
	assert.Contains(t, toolUses[1].Input, "git commit -m")
}

func TestFakeModelCompletesAfterToolResults(t *testing.T) {
	m := NewFakeModel("fake", "fake-1")
	turn, err := m.Stream(context.Background(), Request{
		Messages: []Message{
			UserMessage("commit"),
			{Role: RoleAssistant, Blocks: []Block{{Type: BlockToolUse, ToolName: "git", ToolCallID: "x", Input: `{"command":"git add -a"}`}}},
			{Role: RoleTool, Blocks: []Block{{Type: BlockToolResult, ToolCallID: "x", Text: "ok"}}},
		},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, StopEnd, turn.StopReason)
	assert.Empty(t, toolUseBlocks(turn.Blocks))
}

func TestFakeModelNonCommitPromptStreamsText(t *testing.T) {
	m := NewFakeModel("fake", "fake-1")
	turn, err := m.Stream(context.Background(), Request{
		Messages: []Message{UserMessage("tell me a story")},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, StopEnd, turn.StopReason)
	assert.Empty(t, toolUseBlocks(turn.Blocks))
}

func TestScenario1Count(t *testing.T) {
	for _, tc := range []struct {
		prompt string
		count  int
		ok     bool
	}{
		{"do [scenario1 count=3] please", 3, true},
		{"[scenario1]", 1, true},
		{"[scenario1 count=0]", 1, true}, // non-positive count falls back to default
		{"nothing here", 0, false},
	} {
		count, ok := scenario1Count([]Message{UserMessage(tc.prompt)})
		assert.Equal(t, tc.ok, ok, tc.prompt)
		assert.Equal(t, tc.count, count, tc.prompt)
	}
}

func TestFakeModelScenario1RequestsWriteThenRead(t *testing.T) {
	m := NewFakeModel("fake", "fake-1")
	turn, err := m.Stream(context.Background(), Request{
		Messages: []Message{UserMessage("[scenario1 count=3]")},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, StopToolUse, turn.StopReason)

	// One text block followed by a write then a read tool call.
	require.Len(t, turn.Blocks, 3)
	assert.Equal(t, BlockText, turn.Blocks[0].Type)

	tools := toolUseBlocks(turn.Blocks)
	require.Len(t, tools, 2)
	assert.Equal(t, "create", tools[0].ToolName)
	assert.Contains(t, tools[0].Input, "file_text")
	assert.Equal(t, "cat", tools[1].ToolName)
}

func TestFakeModelScenario1CompletesAfterCount(t *testing.T) {
	m := NewFakeModel("fake", "fake-1")

	// Simulate three completed iterations: each adds one tool-result turn.
	messages := []Message{UserMessage("[scenario1 count=3]")}
	for i := 0; i < 3; i++ {
		messages = append(messages,
			Message{Role: RoleAssistant, Blocks: []Block{{Type: BlockToolUse, ToolName: "cat", ToolCallID: "x"}}},
			Message{Role: RoleTool, Blocks: []Block{{Type: BlockToolResult, ToolCallID: "x", Text: "ok"}}},
		)
	}

	turn, err := m.Stream(context.Background(), Request{Messages: messages}, nil)
	require.NoError(t, err)
	assert.Equal(t, StopEnd, turn.StopReason)
	assert.Empty(t, toolUseBlocks(turn.Blocks))
}

func TestFakeModelScenario1DrivesAgent(t *testing.T) {
	agent := NewAgent(NewFakeModel("fake", "fake-1"), "", []Tool{
		NewTool[catScenarioInput]("create", "write", func(ctx context.Context, in catScenarioInput) (string, error) {
			return "created " + in.Path, nil
		}),
		NewTool[catScenarioInput]("cat", "read", func(ctx context.Context, in catScenarioInput) (string, error) {
			return "content of " + in.Path, nil
		}),
	})

	var toolCalls int
	res, err := agent.Run(context.Background(), "[scenario1 count=2]", RunOptions{
		OnToolCall: func(name, input string) { toolCalls++ },
	})
	require.NoError(t, err)
	assert.Contains(t, res.Text, "completed 2")
	assert.Equal(t, 4, toolCalls) // 2 iterations * (create + cat)
}

// catScenarioInput is a minimal tool input used by the scenario agent test.
type catScenarioInput struct {
	Path     string `json:"path"`
	FileText string `json:"file_text"`
}
