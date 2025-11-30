package templates

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/flosch/pongo2"
	"github.com/pkg/errors"
)

func init() {
	pongo2.RegisterTag("shell", shellFilter)
}

func shellFilter(doc *pongo2.Parser, start *pongo2.Token, arguments *pongo2.Parser) (pongo2.INodeTag, *pongo2.Error) {
	shellNode := &tagShellNode{}

	// Parse until we find the {% endshell %} tag
	wrapper, tagArgs, err := doc.WrapUntilTag("endshell")
	if err != nil {
		return nil, err
	}
	shellNode.wrapper = wrapper

	// Check for any arguments after endshell (currently ignored)
	if tagArgs.Count() > 0 {
		// Arguments after endshell are ignored for now
	}

	return shellNode, nil
}

type tagShellNode struct {
	wrapper *pongo2.NodeWrapper
}

func (node *tagShellNode) Execute(ctx *pongo2.ExecutionContext, writer pongo2.TemplateWriter) *pongo2.Error {
	// Create a temporary buffer to capture the script content
	scriptBuffer := &bytes.Buffer{}

	// Execute the wrapped content to get the script
	err := node.wrapper.Execute(ctx, scriptBuffer)
	if err != nil {
		return err
	}

	scriptContent := scriptBuffer.String()

	// Create a temporary file for the script
	tmpFile, osErr := os.CreateTemp("", "shell-*.sh")
	if osErr != nil {
		return ctx.Error(osErr.Error(), nil)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write the script content to the file
	if _, osErr := tmpFile.WriteString(scriptContent); osErr != nil {
		return ctx.Error(osErr.Error(), nil)
	}

	// Make the file executable
	if osErr := tmpFile.Chmod(0755); osErr != nil {
		return ctx.Error(osErr.Error(), nil)
	}

	// Close the file before executing
	tmpFile.Close()

	// Execute the script using bash
	cmd := exec.Command("bash", tmpFile.Name())
	output, osErr := cmd.Output()
	if osErr != nil {
		// Include stderr if available
		if exitErr, ok := osErr.(*exec.ExitError); ok {
			return ctx.Error(string(exitErr.Stderr), nil)
		}
		return ctx.Error(osErr.Error(), nil)
	}

	// Write the output to the template
	writer.WriteString(string(output))

	return nil
}

// RenderPongo2 renders a template string using the pongo2 engine.
func RenderPongo2(tmplStr string, data map[string]any) (Render, error) {
	ctx := pongo2.Context{}
	for k, v := range data {
		ctx[k] = v
	}
	tpl, err := pongo2.FromString(tmplStr)
	if err != nil {
		return Render{}, errors.WithStack(err)
	}

	systemPrompt, err := tpl.ExecuteBlocks(ctx, []string{"system"})
	if err != nil {
		return Render{}, errors.WithStack(err)
	}
	out, err := tpl.Execute(ctx)
	if err != nil {
		return Render{}, errors.WithStack(err)
	}
	render := Render{
		Prompt: out,
		System: systemPrompt["system"],
	}
	return render, nil
}
