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

func TestIncludeTag(t *testing.T) {
	// Test with relative path
	tmplStr := `{% include "test_include.txt" %}`
	data := map[string]any{}
	render, err := RenderPongo2(tmplStr, data)
	if err != nil {
		t.Fatalf("RenderPongo2 failed: %v", err)
	}
	assert.Contains(t, render.Prompt, "This is test content from an external file.", "Include tag failed: %q", render.Prompt)
	assert.Contains(t, render.Prompt, "It contains multiple lines.", "Include tag missing content: %q", render.Prompt)
	assert.Contains(t, render.Prompt, "Line 3.", "Include tag missing content: %q", render.Prompt)
}

func TestIncludeTagAbsolutePath(t *testing.T) {
	// Test with absolute path
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	absPath := wd + "/test_include.txt"
	tmplStr := `{% include "` + absPath + `" %}`
	data := map[string]any{}
	render, err := RenderPongo2(tmplStr, data)
	if err != nil {
		t.Fatalf("RenderPongo2 failed: %v", err)
	}
	assert.Contains(t, render.Prompt, "This is test content from an external file.", "Include tag with absolute path failed: %q", render.Prompt)
}

func TestIncludeTagFileNotFound(t *testing.T) {
	// Test with non-existent file
	tmplStr := `{% include "non_existent_file.txt" %}`
	data := map[string]any{}
	_, err := RenderPongo2(tmplStr, data)
	assert.Error(t, err, "Expected error for non-existent file")
}
