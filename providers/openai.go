package providers

import (
	"context"
	"github.com/elek/rai/config"
	"reflect"
	"strings"

	"github.com/elek/rai/schema"
	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
)

// OpenAIModel implements the schema.Model interface for OpenAI API
type OpenAIModel struct {
	client *openai.Client
}

// NewOpenAIModel creates a new OpenAI model client
func NewOpenAIModel(p config.Provider) *OpenAIModel {

	client := openai.NewClient(p.Key)
	return &OpenAIModel{
		client: client,
	}
}

// Invoke sends a conversation to the OpenAI API and returns the response
func (o *OpenAIModel) Invoke(ctx context.Context, model config.Model, c *schema.Conversation, tools []schema.Tool) ([]schema.Message, schema.Usage, error) {
	// Prepare messages
	var messages []openai.ChatCompletionMessage

	// Add system message if present
	if c.System != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: c.System,
		})
	}

	// Add conversation messages
	for _, msg := range c.Messages {
		role := msg.Role
		// Map roles to OpenAI format
		switch role {
		case "user":
			role = openai.ChatMessageRoleUser
		case "assistant":
			role = openai.ChatMessageRoleAssistant
		case "system":
			role = openai.ChatMessageRoleSystem
		}

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Prepare tools if any
	var openAITools []openai.Tool
	if len(tools) > 0 {
		for _, tool := range tools {
			// Create function definition
			functionDef := openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
			}

			// Create parameters schema
			params := make(map[string]interface{})
			properties := make(map[string]interface{})
			var required []string

			callbackType := reflect.TypeOf(tool.Callback)
			if callbackType.Kind() == reflect.Func && callbackType.NumIn() > 0 {
				inputType := callbackType.In(0)
				if inputType.Kind() == reflect.Struct {
					for i := 0; i < inputType.NumField(); i++ {
						field := inputType.Field(i)

						// Get JSON name from tag
						jsonTag := field.Tag.Get("json")
						if jsonTag == "" {
							continue
						}
						jsonName := strings.Split(jsonTag, ",")[0]

						// Get description from tag
						description := field.Tag.Get("description")

						// Add to properties
						propDef := map[string]interface{}{
							"type":        getJSONSchemaType(field.Type),
							"description": description,
						}
						properties[jsonName] = propDef

						// Add to required fields if not optional
						required = append(required, jsonName)
					}
				}
			}

			params["type"] = "object"
			params["properties"] = properties
			if len(required) > 0 {
				params["required"] = required
			}

			functionDef.Parameters = params

			openAITools = append(openAITools, openai.Tool{
				Type:     openai.ToolTypeFunction,
				Function: &functionDef,
			})
		}
	}

	// Create request
	req := openai.ChatCompletionRequest{
		Model:       model.Model,
		Messages:    messages,
		MaxTokens:   model.MaxToken,
		Temperature: float32(model.Temperature),
	}

	// Add tools if any
	if len(openAITools) > 0 {
		req.Tools = openAITools
	}

	// Make the API call
	resp, err := o.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, schema.Usage{}, errors.WithStack(err)
	}

	// Process response
	var responseMessages []schema.Message

	// Extract usage information
	usage := schema.Usage{
		InputTokens:  0,
		OutputTokens: 0,
	}

	for _, choice := range resp.Choices {
		message := schema.Message{
			Role:    "assistant",
			Content: choice.Message.Content,
		}

		// Handle tool calls if any
		if len(choice.Message.ToolCalls) > 0 {
			for _, toolCall := range choice.Message.ToolCalls {
				if toolCall.Type == openai.ToolTypeFunction {
					message.ToolName = toolCall.Function.Name
					message.ToolID = toolCall.ID
					message.Content = toolCall.Function.Arguments
				}
			}
		}

		responseMessages = append(responseMessages, message)
	}

	// Update usage information from the response
	usage.InputTokens = resp.Usage.PromptTokens
	usage.OutputTokens = resp.Usage.CompletionTokens

	return responseMessages, usage, nil
}

// ListModels returns a list of available models from OpenAI
func (o *OpenAIModel) ListModels(ctx context.Context) ([]schema.ModelVersion, error) {
	models, err := o.client.ListModels(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var result []schema.ModelVersion
	for _, model := range models.Models {
		result = append(result, schema.ModelVersion{
			ID:   "openai/" + model.ID,
			Name: model.ID,
		})
	}

	return result, nil
}

// Ensure OpenAIModel implements the Model interface
var _ schema.Model = (*OpenAIModel)(nil)
