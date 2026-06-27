package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// ToolInfo describes a tool to the model: its name, description, and the JSON
// schema of its input parameters.
type ToolInfo struct {
	Name        string
	Description string
	// Parameters maps each parameter name to its JSON-schema fragment, e.g.
	// {"city": {"type": "string", "description": "..."}}.
	Parameters map[string]any
	// Required lists the names of required parameters.
	Required []string
}

// ToolCall is a request from the model to invoke a tool.
type ToolCall struct {
	ID    string
	Name  string
	Input string // raw JSON arguments
}

// ToolResult is the outcome of a tool invocation, sent back to the model.
type ToolResult struct {
	Content string
	IsError bool
}

// Tool is a capability the model may invoke during an agent run.
type Tool interface {
	Info() ToolInfo
	Run(ctx context.Context, call ToolCall) (ToolResult, error)
}

// typedTool adapts a strongly-typed callback into a Tool, generating the input
// schema from the struct tags of T via reflection.
type typedTool[T any] struct {
	info ToolInfo
	fn   func(ctx context.Context, in T) (string, error)
}

// NewTool builds a Tool from a typed callback. The JSON schema of the tool's
// parameters is derived from T's `json` and `description` struct tags. On Run,
// the raw JSON input is unmarshaled into T before fn is called. Errors returned
// by fn (or from malformed input) are surfaced as ToolResult.IsError rather than
// as Go errors, so the agent loop can feed them back to the model.
func NewTool[T any](name, description string, fn func(ctx context.Context, in T) (string, error)) Tool {
	params, required := schemaOf(reflect.TypeOf(*new(T)))
	return &typedTool[T]{
		info: ToolInfo{
			Name:        name,
			Description: description,
			Parameters:  params,
			Required:    required,
		},
		fn: fn,
	}
}

func (t *typedTool[T]) Info() ToolInfo { return t.info }

func (t *typedTool[T]) Run(ctx context.Context, call ToolCall) (ToolResult, error) {
	var in T
	if call.Input != "" {
		if err := json.Unmarshal([]byte(call.Input), &in); err != nil {
			return ToolResult{Content: "invalid tool input: " + err.Error(), IsError: true}, nil
		}
	}
	out, err := t.fn(ctx, in)
	if err != nil {
		return ToolResult{Content: err.Error(), IsError: true}, nil
	}
	return ToolResult{Content: out}, nil
}

// schemaOf builds JSON-schema property fragments and the required list for a
// struct type, using its `json` and `description` field tags.
func schemaOf(t reflect.Type) (map[string]any, []string) {
	props := map[string]any{}
	var required []string
	if t == nil || t.Kind() != reflect.Struct {
		return props, required
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name := f.Name
		if tag := f.Tag.Get("json"); tag != "" {
			if n := strings.Split(tag, ",")[0]; n != "" && n != "-" {
				name = n
			} else if n == "-" {
				continue
			}
		}
		prop := map[string]any{"type": jsonType(f.Type)}
		if desc := f.Tag.Get("description"); desc != "" {
			prop["description"] = desc
		}
		props[name] = prop
		required = append(required, name)
	}
	return props, required
}

// jsonType maps a Go type to its JSON-schema type name.
func jsonType(t reflect.Type) string {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	default:
		return "string"
	}
}

// describeTools renders a short human-readable list of tool names, used for the
// DryRun output and debugging.
func describeTools(tools []Tool) string {
	var b strings.Builder
	for _, t := range tools {
		fmt.Fprintf(&b, "   * %s\n", t.Info().Name)
	}
	return b.String()
}
