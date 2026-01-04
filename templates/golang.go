package templates

import (
	"bytes"
	"context"
	"encoding/xml"
	"html/template"
	"io"
	"os"
	"os/exec"
	"strings"

	"charm.land/fantasy"
	"github.com/elek/rai/tool"
	"github.com/google/shlex"

	"github.com/elek/rai/llm"
	"github.com/pkg/errors"
)

func GoTemplateRender(ctx context.Context, tmplStr string, data map[string]any, cb llm.AgentCallback) (string, error) {
	prepared, err := render(tmplStr, data)
	if err != nil {
		return "", errors.WithStack(err)
	}

	reader := strings.NewReader("<root>" + prepared + "</root>")
	decoder := xml.NewDecoder(reader)

	var elementStack []xml.StartElement
	var textStack []string
	var system, prompt, model string
	var enabledTools []fantasy.AgentTool

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", errors.WithStack(err)
		}
		allTools := tool.AllTools()

		switch t := token.(type) {
		case xml.StartElement:
			elementStack = append(elementStack, t)
			textStack = append(textStack, "")
		case xml.EndElement:
			scope := elementStack[len(elementStack)-1]
			content := textStack[len(textStack)-1]
			switch scope.Name.Local {
			case "root":
				prompt = textStack[len(textStack)-1]
				break
			case "system":
				system = textStack[len(textStack)-1]
			case "tool":
				name := getAttr(scope.Attr, "name")
				for _, tl := range allTools {
					if tl.Info().Name == name {
						enabledTools = append(enabledTools, tl)
					}
				}
			case "mcp":
				cmd := getAttr(scope.Attr, "command")
				parts, err := shlex.Split(cmd)
				if err != nil {
					return "", errors.WithStack(err)
				}
				agentTools, close, err := tool.NewMcpAgentTool(ctx, parts[0], parts[1:])
				defer close()
				if err != nil {
					return "", errors.WithStack(err)
				}
				for _, t := range agentTools {
					enabledTools = append(enabledTools, t)
				}
			case "lsp":
				cmd := getAttr(scope.Attr, "command")
				agentTools, close, err := tool.NewLSPAgentTool(ctx, cmd, ".")
				defer close()
				if err != nil {
					return "", errors.WithStack(err)
				}
				for _, t := range agentTools {
					enabledTools = append(enabledTools, t)
				}
			case "exec":
				cmd := getAttr(scope.Attr, "command")
				s, err := execFunc(cmd)
				if err != nil {
					return "", errors.WithStack(err)
				}
				textStack[len(textStack)-2] = textStack[len(textStack)-2] + s
			case "shell":
				textStack[len(textStack)-2] = textStack[len(textStack)-2] + content
			default:
				return "", errors.New("unexpected XML element: " + scope.Name.Local)
			}

			if len(elementStack) > 0 {
				elementStack = elementStack[:len(elementStack)-1]
			}
			if len(textStack) > 0 {
				textStack = textStack[:len(textStack)-1]
			}

		case xml.CharData:

			content := string(t)
			if len(content) > 0 {
				textStack[len(textStack)-1] = textStack[len(textStack)-1] + content
			}
		}
	}

	response, err := cb(ctx, model, system, prompt, enabledTools)
	return response, err
}

func getAttr(attr []xml.Attr, s string) string {
	for _, a := range attr {
		if a.Name.Local == s {
			return a.Value
		}
	}
	return ""
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

func bashFunc(cmd string) (string, error) {
	out, err := exec.Command("bash", "-c", cmd).CombinedOutput()
	return string(out), err
}
