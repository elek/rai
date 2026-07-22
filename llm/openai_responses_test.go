package llm

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/openai/openai-go/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unmarshalInputItems marshals the responses input list and re-decodes it into a
// generic slice so tests can assert on the wire shape without depending on SDK
// param field names.
func unmarshalInputItems(t *testing.T, input responses.ResponseInputParam) []map[string]any {
	t.Helper()
	data, err := json.Marshal(input)
	require.NoError(t, err)
	var items []map[string]any
	require.NoError(t, json.Unmarshal(data, &items))
	return items
}

func TestToResponsesInputRoundTripsReasoningToolAndResult(t *testing.T) {
	msgs := []Message{
		UserMessage("review HEAD"),
		{Role: RoleAssistant, Blocks: []Block{
			{Type: BlockReasoning, ToolCallID: "rs_1", Text: "ENCRYPTED"},
			{Type: BlockToolUse, ToolCallID: "call_1", ToolName: "git", Input: `{"command":"git log"}`},
		}},
		{Role: RoleTool, Blocks: []Block{
			{Type: BlockToolResult, ToolCallID: "call_1", Text: "commit abc"},
		}},
	}

	items := unmarshalInputItems(t, toResponsesInput("be brief", msgs))
	require.Len(t, items, 5)

	// System prompt first, then the user turn.
	assert.Equal(t, "system", items[0]["role"])
	assert.Equal(t, "be brief", items[0]["content"])
	assert.Equal(t, "user", items[1]["role"])
	assert.Equal(t, "review HEAD", items[1]["content"])

	// Reasoning item must precede its function call, carrying id + encrypted content.
	assert.Equal(t, "reasoning", items[2]["type"])
	assert.Equal(t, "rs_1", items[2]["id"])
	assert.Equal(t, "ENCRYPTED", items[2]["encrypted_content"])

	// The tool call maps to a function_call keyed by call_id.
	assert.Equal(t, "function_call", items[3]["type"])
	assert.Equal(t, "call_1", items[3]["call_id"])
	assert.Equal(t, "git", items[3]["name"])
	assert.Equal(t, `{"command":"git log"}`, items[3]["arguments"])

	// The tool result maps to a function_call_output referencing the same call_id.
	assert.Equal(t, "function_call_output", items[4]["type"])
	assert.Equal(t, "call_1", items[4]["call_id"])
	assert.Equal(t, "commit abc", items[4]["output"])
}

func TestToResponsesInputPreservesAssistantTextAlongsideReasoningAndTool(t *testing.T) {
	msgs := []Message{
		UserMessage("do it"),
		{Role: RoleAssistant, Blocks: []Block{
			{Type: BlockText, Text: "Let me check the log."},
			{Type: BlockReasoning, ToolCallID: "rs_1", Text: "ENC"},
			{Type: BlockToolUse, ToolCallID: "call_1", ToolName: "git", Input: "{}"},
		}},
	}

	items := unmarshalInputItems(t, toResponsesInput("", msgs))
	require.Len(t, items, 4)

	assert.Equal(t, "user", items[0]["role"])
	// Assistant text is kept (not dropped) as an assistant-role message, before
	// the reasoning and the tool call it accompanied.
	assert.Equal(t, "assistant", items[1]["role"])
	assert.Equal(t, "Let me check the log.", items[1]["content"])
	assert.Equal(t, "reasoning", items[2]["type"])
	assert.Equal(t, "function_call", items[3]["type"])
}

func TestToResponsesToolsMapsNameDescriptionAndSchema(t *testing.T) {
	type catIn struct {
		Path string `json:"path"`
	}
	tool := NewTool[catIn]("cat", "read a file", func(_ context.Context, _ catIn) (string, error) { return "", nil })

	data, err := json.Marshal(toResponsesTools([]Tool{tool}))
	require.NoError(t, err)
	var arr []map[string]any
	require.NoError(t, json.Unmarshal(data, &arr))

	require.Len(t, arr, 1)
	assert.Equal(t, "function", arr[0]["type"])
	assert.Equal(t, "cat", arr[0]["name"])
	assert.Equal(t, "read a file", arr[0]["description"])
	assert.Contains(t, arr[0], "parameters")
}

func TestToResponsesToolsEmitsArrayRequiredForParamlessTool(t *testing.T) {
	type noArgs struct{}
	tool := NewTool[noArgs]("ping", "pings", func(_ context.Context, _ noArgs) (string, error) { return "", nil })

	data, err := json.Marshal(toResponsesTools([]Tool{tool}))
	require.NoError(t, err)
	var arr []map[string]any
	require.NoError(t, json.Unmarshal(data, &arr))

	params, ok := arr[0]["parameters"].(map[string]any)
	require.True(t, ok, "parameters must be an object, got %#v", arr[0]["parameters"])
	// The Responses API strictly validates the schema: `required` must be an
	// array (JSON []), never null, even when there are no parameters.
	require.Contains(t, params, "required")
	_, isArray := params["required"].([]any)
	assert.True(t, isArray, "required must serialize as a JSON array, got %#v", params["required"])
	_, propsObj := params["properties"].(map[string]any)
	assert.True(t, propsObj, "properties must serialize as a JSON object, got %#v", params["properties"])
}

func TestBlocksFromResponseItemsReasoningThenToolCall(t *testing.T) {
	var reasoning responses.ResponseOutputItemUnion
	require.NoError(t, json.Unmarshal([]byte(`{"type":"reasoning","id":"rs_1","encrypted_content":"ENC","summary":[]}`), &reasoning))
	var fn responses.ResponseOutputItemUnion
	require.NoError(t, json.Unmarshal([]byte(`{"type":"function_call","call_id":"call_1","name":"git","arguments":"{}"}`), &fn))

	blocks, stop := blocksFromResponseItems([]responses.ResponseOutputItemUnion{reasoning, fn})

	require.Len(t, blocks, 2)
	assert.Equal(t, BlockReasoning, blocks[0].Type)
	assert.Equal(t, "rs_1", blocks[0].ToolCallID)
	assert.Equal(t, "ENC", blocks[0].Text)
	assert.Equal(t, BlockToolUse, blocks[1].Type)
	assert.Equal(t, "call_1", blocks[1].ToolCallID)
	assert.Equal(t, "git", blocks[1].ToolName)
	assert.Equal(t, "{}", blocks[1].Input)
	assert.Equal(t, StopToolUse, stop)
}

func TestBlocksFromResponseItemsTextOnly(t *testing.T) {
	var msg responses.ResponseOutputItemUnion
	require.NoError(t, json.Unmarshal([]byte(`{"type":"message","id":"m1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"hello","annotations":[]}]}`), &msg))

	blocks, stop := blocksFromResponseItems([]responses.ResponseOutputItemUnion{msg})

	require.Len(t, blocks, 1)
	assert.Equal(t, BlockText, blocks[0].Type)
	assert.Equal(t, "hello", blocks[0].Text)
	assert.Equal(t, StopEnd, stop)
}
