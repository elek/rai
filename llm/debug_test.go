package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatRequest(t *testing.T) {
	req := Request{
		System:    "you are helpful",
		MaxTokens: 1024,
		Messages: []Message{
			UserMessage("hello"),
			{Role: RoleAssistant, Blocks: []Block{
				TextBlock("let me check"),
				{Type: BlockToolUse, ToolName: "cat", ToolCallID: "t1", Input: `{"path":"a.go"}`},
			}},
			{Role: RoleTool, Blocks: []Block{
				{Type: BlockToolResult, ToolCallID: "t1", Text: "file contents"},
			}},
		},
	}

	out := formatRequest("anthropic", "claude", req)

	assert.Contains(t, out, ">>> anthropic/claude request (max_tokens=1024)")
	assert.Contains(t, out, "[system]")
	assert.Contains(t, out, "you are helpful")
	assert.Contains(t, out, "[user]")
	assert.Contains(t, out, "hello")
	assert.Contains(t, out, "tool_use cat (id=t1)")
	assert.Contains(t, out, `{"path":"a.go"}`)
	assert.Contains(t, out, "tool_result (id=t1)")
	assert.Contains(t, out, "file contents")
}

func TestFormatTurn(t *testing.T) {
	turn := &Turn{
		StopReason: StopToolUse,
		Usage:      Usage{InputTokens: 10, OutputTokens: 20},
		Blocks: []Block{
			TextBlock("done"),
			{Type: BlockToolUse, ToolName: "git", ToolCallID: "g1", Input: "{}"},
		},
	}

	out := formatTurn("openai", "gpt", turn)

	assert.Contains(t, out, "<<< openai/gpt response (stop=tool_use, tokens in=10 out=20)")
	assert.Contains(t, out, "done")
	assert.Contains(t, out, "tool_use git (id=g1)")
}
