package providers

import (
	"context"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/elek/rai/config"
	"github.com/elek/rai/schema"
	"github.com/pkg/errors"
	"reflect"
	"strings"
	"time"
)

type ClaudeModel struct {
	client *anthropic.Client
}

func (c *ClaudeModel) ListModels(ctx context.Context) (res []schema.ModelVersion, err error) {
	list, err := c.client.Models.List(ctx, anthropic.ModelListParams{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for _, l := range list.Data {
		res = append(res, schema.ModelVersion{
			ID:   "anthropic/" + l.ID,
			Name: l.DisplayName,
		})
	}
	return res, nil
}

func NewClaudeModel(p config.Provider) *ClaudeModel {
	client := anthropic.NewClient(
		option.WithAPIKey(p.Key),
	)
	return &ClaudeModel{
		client: &client,
	}
}

// SetMaxTokens sets the maximum number of tokens to generate
func (c *ClaudeModel) SetMaxTokens(maxTokens int) {
	// This will be used in Invoke
}

// SetTemperature sets the temperature for generation
func (c *ClaudeModel) SetTemperature(temperature float64) {
	// This will be used in Invoke
}

// getJSONSchemaType converts Go types to JSON Schema types
func getJSONSchemaType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.String:
		return "string"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	default:
		// Default to string for any other types
		if t == reflect.TypeOf(time.Time{}) {
			return "string"
		}
		return "string"
	}
}

func (c *ClaudeModel) Invoke(ctx context.Context, model config.Model, conversation *schema.Conversation, tools []schema.Tool) ([]schema.Message, schema.Usage, error) {
	req := anthropic.MessageNewParams{
		Model:       model.Model,
		MaxTokens:   int64(model.MaxToken),
		Temperature: param.NewOpt(model.Temperature),
	}
	if conversation.System != "" {
		req.System = append(req.System, anthropic.TextBlockParam{
			Text: conversation.System,
		})
	}

	for _, c := range conversation.Messages {
		if c.Original != nil {
			var cbu = c.Original.(anthropic.ContentBlockUnion)
			if c.Role == "assistant" {
				req.Messages = append(req.Messages, anthropic.NewAssistantMessage(cbu.ToParam()))
			} else if c.Role == "user" {
				req.Messages = append(req.Messages, anthropic.NewUserMessage(cbu.ToParam()))
			}
			continue
		}
		if c.Role == "assistant" {
			req.Messages = append(req.Messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(c.Content)))
		} else if c.Role == "user" {
			req.Messages = append(req.Messages, anthropic.NewUserMessage(anthropic.NewTextBlock(c.Content)))
		} else if c.Role == "tool" {
			req.Messages = append(req.Messages, anthropic.NewUserMessage(anthropic.NewToolResultBlock(c.ToolID, c.Content, false)))

		}

	}

	for _, t := range tools {

		extraFields := map[string]interface{}{}

		// Generate extraFields from the callback function's parameter
		if t.Callback != nil {
			callbackType := reflect.TypeOf(t.Callback)
			// Check if it's a function with at least one parameter
			if callbackType.Kind() == reflect.Func && callbackType.NumIn() > 0 {

				paramType := callbackType.In(0)

				if paramType.Kind() == reflect.Struct {
					for i := 0; i < paramType.NumField(); i++ {
						field := paramType.Field(i)

						fieldSchema := map[string]interface{}{
							"type": getJSONSchemaType(field.Type),
						}

						if desc, ok := field.Tag.Lookup("description"); ok {
							fieldSchema["description"] = desc
						}
						name := field.Name
						if nameTag, ok := field.Tag.Lookup("json"); ok {
							name = strings.Split(nameTag, ",")[0]
						}
						extraFields[name] = fieldSchema
					}
				}
			}
		}

		tp := anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        t.Name,
				Description: anthropic.String(t.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Type:       "object",
					Properties: extraFields,
				},
			},
		}
		req.Tools = append(req.Tools, tp)
	}

	if model.Debug {
		debug("req", req)
	}
	resp, err := c.client.Messages.New(ctx, req)
	if err != nil {
		return nil, schema.Usage{}, errors.WithStack(err)
	}
	if model.Debug {
		debug("resp", resp)
	}
	var m []schema.Message
	for _, cont := range resp.Content {
		switch content := cont.AsAny().(type) {
		case anthropic.TextBlock:
			m = append(m, schema.Message{
				Role:     "assistant",
				Content:  content.Text,
				Original: cont,
			})
		case anthropic.ToolUseBlock:
			m = append(m, schema.Message{
				Role:     "assistant",
				ToolName: content.Name,
				ToolID:   content.ID,
				Content:  string(content.Input),
				Original: cont,
			})
		default:
			panic("unexpected type")
		}
	}

	// Extract usage information from the response
	usage := schema.Usage{
		InputTokens:  int(resp.Usage.InputTokens),
		OutputTokens: int(resp.Usage.OutputTokens),
	}

	return m, usage, nil
}

var _ schema.Model = (*ClaudeModel)(nil)
