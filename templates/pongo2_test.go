package templates

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderPongo2(t *testing.T) {
	tmplBytes, err := os.ReadFile("pongo2_test.txt")
	if err != nil {
		t.Fatalf("failed to read template: %v", err)
	}
	tmplStr := string(tmplBytes)
	data := map[string]any{
		"name":           "World",
		"system_message": "This is a system block.",
		"prompt_body":    "This is the prompt body.",
	}
	render, err := RenderPongo2(tmplStr, data)
	if err != nil {
		t.Fatalf("RenderPongo2 failed: %v", err)
	}
	assert.Contains(t, render.Prompt, "Hello, World!", "Prompt missing name: %q", render.Prompt)
	assert.Contains(t, render.Prompt, "foobar", "Prompt missing shell output: %q", render.Prompt)
	assert.Contains(t, render.Prompt, "This is the prompt body.", "Prompt missing prompt_body: %q", render.Prompt)
	assert.Contains(t, render.System, "This is a system block.", "System missing system_message: %q", render.System)
}
