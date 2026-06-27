package templates

import (
	"context"
	"testing"

	"github.com/elek/rai/config"
	"github.com/elek/rai/llm"
	"github.com/stretchr/testify/require"
)

func TestRender(t *testing.T) {
	inp := `
foobar
<system>
system prompt
</system>
<shell>
echo "asd"
</shell>
`
	var capturedSystem string
	callback := func(ctx context.Context, model config.Model, system string, prompt string, tools []llm.Tool) (string, error) {
		capturedSystem = system
		return "test response", nil
	}

	cfg := config.Config{}
	response, err := GoTemplateRender(cfg)(context.Background(), inp, map[string]any{}, callback)
	require.NoError(t, err)
	require.Equal(t, "test response", response)
	require.Equal(t, "\nsystem prompt\n", capturedSystem)
}

func TestModuleSupport(t *testing.T) {
	testCfg := config.Config{
		Models: []config.Model{
			{Name: "gpt-4", Provider: "openrouter", Model: "gpt-4"},
			{Name: "claude-3-opus", Provider: "anthropic", Model: "claude-3-opus"},
			{Name: "gemini-pro", Provider: "google", Model: "gemini-pro"},
		},
	}

	tests := []struct {
		name           string
		template       string
		expectedModel  string
		expectedSystem string
		expectedPrompt string
	}{
		{
			name: "model with model name",
			template: `
<model>gpt-4</model>
<system>system message</system>
user prompt here
`,
			expectedModel:  "gpt-4",
			expectedSystem: "system message",
			expectedPrompt: "\n\n\nuser prompt here\n",
		},
		{
			name: "model with whitespace",
			template: `
<model>
  claude-3-opus
</model>
<system>another system</system>
another prompt
`,
			expectedModel:  "claude-3-opus",
			expectedSystem: "another system",
			expectedPrompt: "\n\n\nanother prompt\n",
		},
		{
			name: "no model specified",
			template: `
<system>no model system</system>
no model prompt
`,
			expectedModel:  "",
			expectedSystem: "no model system",
			expectedPrompt: "\n\nno model prompt\n",
		},
		{
			name: "model only",
			template: `
<model>gemini-pro</model>
just a prompt
`,
			expectedModel:  "gemini-pro",
			expectedSystem: "",
			expectedPrompt: "\n\njust a prompt\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedModel config.Model
			var capturedSystem, capturedPrompt string
			callback := func(ctx context.Context, model config.Model, system string, prompt string, tools []llm.Tool) (string, error) {
				capturedModel = model
				capturedSystem = system
				capturedPrompt = prompt
				return "response", nil
			}

			response, err := GoTemplateRender(testCfg)(context.Background(), tt.template, map[string]any{}, callback)
			require.NoError(t, err)
			require.Equal(t, "response", response)
			require.Equal(t, tt.expectedModel, capturedModel.Name, "model mismatch")
			require.Equal(t, tt.expectedSystem, capturedSystem, "system mismatch")
			require.Equal(t, tt.expectedPrompt, capturedPrompt, "prompt mismatch")
		})
	}
}
