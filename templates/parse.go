package templates

import (
	"context"
	"encoding/xml"
	"io"
	"strings"

	"github.com/elek/rai/config"
	"github.com/elek/rai/llm"
	"github.com/elek/rai/tool"
	"github.com/google/shlex"
	"github.com/pkg/errors"
)

// ParsedTemplate holds the parsed elements of a template, including the model,
// system prompt, user prompt, and any agent tools that were configured.
type ParsedTemplate struct {
	Model   config.Model
	System  string
	Prompt  string
	Tools   []llm.Tool
	Closers []func()
}

// Close releases any resources held by the parsed template, such as MCP and LSP connections.
func (p *ParsedTemplate) Close() {
	for _, c := range p.Closers {
		c()
	}
}

// ParseTemplate renders the given template string with the provided data, then parses
// the XML structure to extract model, system prompt, user prompt, and tool configurations.
// The caller is responsible for calling Close() on the returned ParsedTemplate.
func ParseTemplate(ctx context.Context, cfg config.Config, tmplStr string, data map[string]any) (*ParsedTemplate, error) {
	prepared, err := render(tmplStr, data)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	reader := strings.NewReader("<root>" + prepared + "</root>")
	decoder := xml.NewDecoder(reader)

	var elementStack []xml.StartElement
	var textStack []string
	result := &ParsedTemplate{}

	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, errors.WithStack(err)
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
				result.Prompt = textStack[len(textStack)-1]
			case "system":
				result.System = textStack[len(textStack)-1]
			case "model":
				modelName := strings.TrimSpace(textStack[len(textStack)-1])
				mod, found := cfg.FindModel(modelName)
				if !found {
					return nil, errors.New("model couldn't be found: " + modelName)
				}
				result.Model = mod
			case "tool":
				name := getAttr(scope.Attr, "name")
				for _, tl := range allTools {
					if tl.Info().Name == name {
						result.Tools = append(result.Tools, tl)
					}
				}
			case "mcp":
				cmd := getAttr(scope.Attr, "command")
				parts, err := shlex.Split(cmd)
				if err != nil {
					return nil, errors.WithStack(err)
				}
				agentTools, closer, err := tool.NewMcpAgentTool(ctx, parts[0], parts[1:])
				if err != nil {
					return nil, errors.WithStack(err)
				}
				result.Closers = append(result.Closers, closer)
				result.Tools = append(result.Tools, agentTools...)
			case "lsp":
				cmd := getAttr(scope.Attr, "command")
				agentTools, closer, err := tool.NewLSPAgentTool(ctx, cmd, ".")
				if err != nil {
					return nil, errors.WithStack(err)
				}
				result.Closers = append(result.Closers, closer)
				result.Tools = append(result.Tools, agentTools...)
			case "exec":
				cmd := getAttr(scope.Attr, "command")
				s, err := execFunc(cmd)
				if err != nil {
					return nil, errors.WithStack(err)
				}
				textStack[len(textStack)-2] = textStack[len(textStack)-2] + s
			case "shell":
				textStack[len(textStack)-2] = textStack[len(textStack)-2] + content
			default:
				return nil, errors.New("unexpected XML element: " + scope.Name.Local)
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

	return result, nil
}

func getAttr(attr []xml.Attr, s string) string {
	for _, a := range attr {
		if a.Name.Local == s {
			return a.Value
		}
	}
	return ""
}
