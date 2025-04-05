package tool

import (
	"github.com/elek/rai/schema"
	"github.com/stretchr/testify/require"
	"testing"
)

type HelloWorldInput struct {
	Name string `json:"name"`
}

func HelloWorld(input HelloWorldInput) string {
	return "Hello " + input.Name

}

func TestHandleTools(t *testing.T) {
	msgs := []schema.Message{
		{
			Role:     "assistant",
			Content:  `{"name": "world"}`,
			ToolName: "hello_world",
			ToolID:   "123",
		},
	}

	tools := []schema.Tool{
		{
			Name:        "hello_world",
			Description: "Hello world tool",
			Callback:    HelloWorld,
		},
	}

	rmsgs, handled, err := HandleTools(msgs, tools)
	require.NoError(t, err)
	require.Len(t, rmsgs, 2)
	require.True(t, handled)
}
