package templates

import (
	"bytes"
	"context"
	"html/template"
	"os"
	"os/exec"
	"strings"

	"github.com/elek/rai/config"
	"github.com/elek/rai/llm"
	"github.com/google/shlex"
)

// GoTemplateRender returns a function that parses a template string and invokes the
// callback with the extracted model, system prompt, user prompt, and tools.
func GoTemplateRender(cfg config.Config) func(ctx context.Context, tmplStr string, data map[string]any, cb llm.AgentCallback) (string, error) {
	return func(ctx context.Context, tmplStr string, data map[string]any, cb llm.AgentCallback) (string, error) {
		parsed, err := ParseTemplate(ctx, cfg, tmplStr, data)
		if err != nil {
			return "", err
		}
		defer parsed.Close()

		response, err := cb(ctx, parsed.Model, parsed.System, parsed.Prompt, parsed.Tools)
		return response, err
	}
}

func render(tmplStr string, data map[string]any) (string, error) {
	templateDef := template.New("prompt")
	tpl, err := templateDef.Parse(tmplStr)
	if err != nil {
		return "", err
	}
	out := bytes.NewBuffer([]byte{})
	err = tpl.Execute(out, data)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

// RunFunc executes a command, sending stdout and stderr to the current process outputs.
func RunFunc(cmd string) (string, error) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "", nil
	}
	c := exec.Command(parts[0], parts[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	return "", err
}

func execFunc(cmd string) (string, error) {
	parts, err := shlex.Split(cmd)
	if err != nil {
		return "", err
	}
	if len(parts) == 0 {
		return "", nil
	}
	out, err := exec.Command(parts[0], parts[1:]...).CombinedOutput()
	return string(out), err
}
