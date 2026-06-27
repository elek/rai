package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type weatherInput struct {
	City string `json:"city" description:"The city to look up"`
	Days int    `json:"days" description:"Number of days"`
}

func TestNewToolBuildsSchemaFromTags(t *testing.T) {
	tl := NewTool[weatherInput]("weather", "Get the weather", func(ctx context.Context, in weatherInput) (string, error) {
		return "", nil
	})

	info := tl.Info()
	assert.Equal(t, "weather", info.Name)
	assert.Equal(t, "Get the weather", info.Description)

	require.Contains(t, info.Parameters, "city")
	city := info.Parameters["city"].(map[string]any)
	assert.Equal(t, "string", city["type"])
	assert.Equal(t, "The city to look up", city["description"])

	require.Contains(t, info.Parameters, "days")
	days := info.Parameters["days"].(map[string]any)
	assert.Equal(t, "integer", days["type"])

	assert.ElementsMatch(t, []string{"city", "days"}, info.Required)
}

func TestNewToolRunUnmarshalsInput(t *testing.T) {
	var got weatherInput
	tl := NewTool[weatherInput]("weather", "Get the weather", func(ctx context.Context, in weatherInput) (string, error) {
		got = in
		return "sunny in " + in.City, nil
	})

	res, err := tl.Run(context.Background(), ToolCall{Name: "weather", Input: `{"city":"Madrid","days":3}`})
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, "sunny in Madrid", res.Content)
	assert.Equal(t, "Madrid", got.City)
	assert.Equal(t, 3, got.Days)
}

func TestNewToolRunReportsErrorAsResult(t *testing.T) {
	tl := NewTool[weatherInput]("weather", "Get the weather", func(ctx context.Context, in weatherInput) (string, error) {
		return "", assert.AnError
	})

	res, err := tl.Run(context.Background(), ToolCall{Name: "weather", Input: `{"city":"Madrid"}`})
	require.NoError(t, err) // tool errors surface as IsError results, not Go errors
	assert.True(t, res.IsError)
	assert.Contains(t, res.Content, assert.AnError.Error())
}

func TestNewToolRunRejectsBadJSON(t *testing.T) {
	tl := NewTool[weatherInput]("weather", "Get the weather", func(ctx context.Context, in weatherInput) (string, error) {
		return "ok", nil
	})

	res, err := tl.Run(context.Background(), ToolCall{Name: "weather", Input: `{not json`})
	require.NoError(t, err)
	assert.True(t, res.IsError)
}
