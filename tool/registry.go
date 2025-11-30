package tool

import (
	"reflect"

	"github.com/tmc/langchaingo/llms"
)

type ToolDef struct {
	Name        string
	Description string
	Callback    any
}

func AllTools() (res []ToolDef) {
	res = append(res, ToolDef{
		Name:        "cat",
		Description: "Read file content with optional offset and line limits",
		Callback:    Cat,
	})
	res = append(res, ToolDef{
		Name:        "files",
		Description: "List files in a directory, with options for recursive listing and pattern matching",
		Callback:    ListFiles,
	})
	res = append(res, ToolDef{
		Name:        "git",
		Description: "Execute any git command in the local repository",
		Callback:    Git,
	})

	return res
}

func AsFunction(tools []ToolDef) (res []llms.Tool) {
	for _, t := range tools {

		properties := map[string]any{}
		ProcessParams(t.Callback, func(name string, t reflect.Type, desc string) {
			typeStr := "string"
			switch t {
			case reflect.TypeOf(int(0)), reflect.TypeOf(int8(0)), reflect.TypeOf(int16(0)), reflect.TypeOf(int32(0)), reflect.TypeOf(int64(0)):
				typeStr = "integer"
			case reflect.TypeOf(true):
				typeStr = "boolean"
			case reflect.TypeOf(""):
				typeStr = "string"
			default:
				panic("unsupported type " + t.String())
			}
			properties[name] = map[string]any{
				"type":        typeStr,
				"description": desc,
			}
		})

		res = append(res, llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters: map[string]any{
					"type":       "object",
					"properties": properties,
				},
			},
		})
	}
	return res
}
