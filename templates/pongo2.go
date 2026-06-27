package templates

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/flosch/pongo2"
	"github.com/pkg/errors"
)

func init() {
	if err := pongo2.RegisterTag("shell", shellFilter); err != nil {
		panic(err)
	}
	// pongo2 ships a built-in "include" tag; ReplaceTag overrides it with our
	// version, which reads an arbitrary file by (absolute) path.
	if err := pongo2.ReplaceTag("include", includeTag); err != nil {
		panic(err)
	}
}

func shellFilter(doc *pongo2.Parser, start *pongo2.Token, arguments *pongo2.Parser) (pongo2.INodeTag, *pongo2.Error) {
	shellNode := &tagShellNode{}

	// Parse until we find the {% endshell %} tag
	wrapper, tagArgs, err := doc.WrapUntilTag("endshell")
	if err != nil {
		return nil, err
	}
	shellNode.wrapper = wrapper

	// Arguments after endshell are currently ignored.
	_ = tagArgs

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
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	defer func() { _ = tmpFile.Close() }()

	// Write the script content to the file
	if _, osErr := tmpFile.WriteString(scriptContent); osErr != nil {
		return ctx.Error(osErr.Error(), nil)
	}

	// Make the file executable
	if osErr := tmpFile.Chmod(0755); osErr != nil {
		return ctx.Error(osErr.Error(), nil)
	}

	// Close the file before executing
	if osErr := tmpFile.Close(); osErr != nil {
		return ctx.Error(osErr.Error(), nil)
	}

	// Execute the script using bash
	cmd := exec.Command("bash", tmpFile.Name())
	output, osErr := cmd.Output()
	if osErr != nil {
		// Include stderr if available
		exitErr := &exec.ExitError{}
		if errors.As(osErr, &exitErr) {
			return ctx.Error(string(exitErr.Stderr), nil)
		}
		return ctx.Error(osErr.Error(), nil)
	}

	// Write the output to the template
	if _, osErr := writer.WriteString(string(output)); osErr != nil {
		return ctx.Error(osErr.Error(), nil)
	}

	return nil
}

func includeTag(doc *pongo2.Parser, start *pongo2.Token, arguments *pongo2.Parser) (pongo2.INodeTag, *pongo2.Error) {
	includeNode := &tagIncludeNode{}

	// Parse the file path argument
	if arguments.Remaining() == 0 {
		return nil, arguments.Error("include tag requires a file path argument", nil)
	}

	// Get the file path expression
	filePathExpr, err := arguments.ParseExpression()
	if err != nil {
		return nil, err
	}
	includeNode.filePath = filePathExpr

	// Check if there are unexpected additional arguments
	if arguments.Remaining() > 0 {
		return nil, arguments.Error("include tag takes only one argument (file path)", nil)
	}

	return includeNode, nil
}

type tagIncludeNode struct {
	filePath pongo2.IEvaluator
}

func (node *tagIncludeNode) Execute(ctx *pongo2.ExecutionContext, writer pongo2.TemplateWriter) *pongo2.Error {
	// Evaluate the file path expression
	filePathValue, err := node.filePath.Evaluate(ctx)
	if err != nil {
		return err
	}

	filePath := filePathValue.String()
	if filePath == "" {
		return ctx.Error("include tag requires a non-empty file path", nil)
	}

	// Convert relative paths to absolute paths
	absPath, osErr := filepath.Abs(filePath)
	if osErr != nil {
		return ctx.Error(osErr.Error(), nil)
	}

	// Read the file content
	content, osErr := os.ReadFile(absPath)
	if osErr != nil {
		return ctx.Error(osErr.Error(), nil)
	}

	// Write the content to the template
	if _, osErr := writer.WriteString(string(content)); osErr != nil {
		return ctx.Error(osErr.Error(), nil)
	}

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
