package providers

import (
	"context"
	"encoding/json"
	"github.com/elek/rai/config"
	"github.com/elek/rai/schema"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"reflect"
	"strings"
)

type OpenRouter struct {
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
}

func NewOpenRouter(p config.Provider) *OpenRouter {
	return &OpenRouter{
		apiKey: p.Key,
	}
}

type openRouterRequest struct {
	Model       string              `json:"model"`
	Messages    []openRouterMessage `json:"messages"`
	Tools       []openRouterTool    `json:"tools,omitempty"`
	MaxTokens   int                 `json:"max_tokens"`
	Temperature float64             `json:"temperature,omitempty"`
	Stream      bool                `json:"stream"`
}

type openRouterMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type openRouterTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type openRouterTool struct {
	Type     string                 `json:"type"`
	Function openRouterToolFunction `json:"function"`
}

type openRouterToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type openRouterResponse struct {
	ID      string                  `json:"id"`
	Choices []openRouterChoice      `json:"choices"`
	Error   *openRouterErrorMessage `json:"error,omitempty"`
	Usage   *openRouterUsage        `json:"usage,omitempty"`
}

type openRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openRouterChoice struct {
	Message openRouterChoiceMessage `json:"message"`
}

type openRouterChoiceMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterResponseContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type openRouterResponseToolCall struct {
	Type     string                         `json:"type"`
	ID       string                         `json:"id"`
	Function openRouterResponseToolFunction `json:"function"`
}

type openRouterResponseToolFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type openRouterErrorMessage struct {
	Message  string                 `json:"message"`
	Code     int                    `json:"code"`
	Metadata map[string]interface{} `json:"metadata"`
}

func (o OpenRouter) Invoke(ctx context.Context, model config.Model, c *schema.Conversation, tools []schema.Tool) ([]schema.Message, schema.Usage, error) {
	// Prepare messages
	var requestMessages []openRouterMessage

	// Add system message if present
	if c.System != "" {
		requestMessages = append(requestMessages, openRouterMessage{
			Role:    "system",
			Content: c.System,
		})
	}

	// Add user and assistant messages
	for _, msg := range c.Messages {
		requestMessages = append(requestMessages, openRouterMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Prepare tools
	var openRouterTools []openRouterTool
	if len(tools) > 0 {
		for _, t := range tools {
			// Generate parameters from the callback function's parameter
			parameters := map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}

			// Extract properties from tool callback
			if t.Callback != nil {
				callbackType := reflect.TypeOf(t.Callback)
				// Check if it's a function with at least one parameter
				if callbackType.Kind() == reflect.Func && callbackType.NumIn() > 0 {
					// Get the first parameter type (expected to be the tool's input schema)
					paramType := callbackType.In(0)

					// If it's a struct, extract field information
					if paramType.Kind() == reflect.Struct {
						properties := map[string]interface{}{}
						for i := 0; i < paramType.NumField(); i++ {
							field := paramType.Field(i)
							// Create schema entry based on field type
							fieldSchema := map[string]interface{}{
								"type": getJSONSchemaType(field.Type),
							}
							// Add description if available in the struct tags
							if desc, ok := field.Tag.Lookup("description"); ok {
								fieldSchema["description"] = desc
							}
							name := field.Name
							if nameTag, ok := field.Tag.Lookup("json"); ok {
								name = strings.Split(nameTag, ",")[0]
							}
							properties[name] = fieldSchema
						}
						parameters["properties"] = properties
					}
				}
			}

			openRouterTools = append(openRouterTools, openRouterTool{
				Type: "function",
				Function: openRouterToolFunction{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  parameters,
				},
			})
		}
	}

	req := openRouterRequest{
		Model:       o.model,
		Messages:    requestMessages,
		MaxTokens:   o.maxTokens,
		Temperature: o.temperature,
		Stream:      false,
	}

	// Add tools if any
	if len(openRouterTools) > 0 {
		req.Tools = openRouterTools
	}

	// Convert request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, schema.Usage{}, errors.WithStack(err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, schema.Usage{}, errors.WithStack(err)
	}

	// Add headers
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, schema.Usage{}, errors.WithStack(err)
	}
	defer resp.Body.Close()

	// Check if request was successful
	if resp.StatusCode != http.StatusOK {
		var errResp openRouterResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, schema.Usage{}, errors.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		if errResp.Error != nil {
			return nil, schema.Usage{}, errors.Errorf("OpenRouter API error: %s", errResp.Error.Message)
		}
		return nil, schema.Usage{}, errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, schema.Usage{}, errors.WithStack(err)
	}
	// Parse response
	var openRouterResp openRouterResponse
	if err := json.Unmarshal([]byte(body), &openRouterResp); err != nil {
		return nil, schema.Usage{}, errors.WithStack(err)
	}

	if openRouterResp.Error != nil {
		return nil, schema.Usage{}, errors.Errorf("error: %s %s", openRouterResp.Error.Message, openRouterResp.Error.Metadata)

	}

	// Convert response to schema.Message
	var responseMessages []schema.Message

	if len(openRouterResp.Choices) > 0 {
		for _, choice := range openRouterResp.Choices {
			switch choice.Message.Role {
			case "assistant":
				responseMessages = append(responseMessages, schema.Message{
					Role:    "assistant",
					Content: choice.Message.Content,
				})
			}
		}
	}

	//// Debug output
	//indent, err := json.MarshalIndent(responseMessages, "", "  ")
	//if err != nil {
	//	return nil, schema.Usage{}, errors.WithStack(err)
	//}
	//fmt.Println(string(indent))

	// Create usage information
	usage := schema.Usage{
		InputTokens:  0,
		OutputTokens: 0,
	}

	// If usage information is available in the response, use it
	if openRouterResp.Usage != nil {
		usage.InputTokens = openRouterResp.Usage.PromptTokens
		usage.OutputTokens = openRouterResp.Usage.CompletionTokens
	}

	return responseMessages, usage, nil
}

type openRouterModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type openRouterModelsResponse struct {
	Data []openRouterModel `json:"data"`
}

func (o OpenRouter) ListModels(ctx context.Context) ([]schema.ModelVersion, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var modelsResp openRouterModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, errors.WithStack(err)
	}

	var models []schema.ModelVersion
	for _, model := range modelsResp.Data {
		models = append(models, schema.ModelVersion{
			ID:   "openrouter/" + model.ID,
			Name: model.Name,
		})
	}

	return models, nil
}

var _ schema.Model = (*OpenRouter)(nil)
