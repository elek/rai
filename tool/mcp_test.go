package tool

import (
	"encoding/json"
	"fmt"
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
)

func TestMcpClient(t *testing.T) {

}

func TestMcp(t *testing.T) {
	tool, f, err := NewMcpAgentTool(t.Context(), "yvp", []string{"stdio"})
	require.NoError(t, err)
	defer f()

	params := map[string]interface{}{
		"url": "https://www.youtube.com/watch?v=MDfGDbKO-ZY",
	}

	encoded, err := json.Marshal(params)
	require.NoError(t, err)

	run, err := tool[0].Run(t.Context(), fantasy.ToolCall{
		ID:    "1",
		Name:  "youtube-text-transcription",
		Input: string(encoded),
	})
	require.NoError(t, err)
	fmt.Println(run.Content)
}
