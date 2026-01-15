package templates

import (
	"context"
	"testing"

	"charm.land/fantasy"
	"github.com/elek/rai/config"
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
	callback := func(ctx context.Context, model config.Model, system string, prompt string, tools []fantasy.AgentTool) (string, error) {
		capturedSystem = system
		return "test response", nil
	}

	response, err := GoTemplateRender(context.Background(), inp, map[string]any{}, callback)
	require.NoError(t, err)
	require.Equal(t, "test response", response)
	require.Equal(t, "\nsystem prompt\n", capturedSystem)
}

func TestModuleSupport(t *testing.T) {
	tests := []struct {
		name           string
		template       string
		expectedModel  string
		expectedSystem string
		expectedPrompt string
	}{
		{
			name: "module with model name",
			template: `
<module>gpt-4</module>
<system>system message</system>
user prompt here
`,
			expectedModel:  "gpt-4",
			expectedSystem: "system message",
			expectedPrompt: "\n\n\nuser prompt here\n",
		},
		{
			name: "module with whitespace",
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
			name: "no module specified",
			template: `
<system>no model system</system>
no model prompt
`,
			expectedModel:  "",
			expectedSystem: "no model system",
			expectedPrompt: "\n\nno model prompt\n",
		},
		{
			name: "module only",
			template: `
<module>gemini-pro</module>
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
			callback := func(ctx context.Context, model config.Model, system string, prompt string, tools []fantasy.AgentTool) (string, error) {
				capturedModel = model
				capturedSystem = system
				capturedPrompt = prompt
				return "response", nil
			}

			response, err := GoTemplateRender(context.Background(), tt.template, map[string]any{}, callback)
			require.NoError(t, err)
			require.Equal(t, "response", response)
			require.Equal(t, tt.expectedModel, capturedModel, "model mismatch")
			require.Equal(t, tt.expectedSystem, capturedSystem, "system mismatch")
			require.Equal(t, tt.expectedPrompt, capturedPrompt, "prompt mismatch")
		})
	}
}
