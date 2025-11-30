package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/tmc/langchaingo/llms"

	"reflect"
)

//messageHistory,err = tool.HandleTools(ctx, llm, tools, messageHistory, resp)

func HandleTools(ctx context.Context, llm llms.Model, tools []ToolDef, messageHistory []llms.MessageContent, resp *llms.ContentResponse) ([]llms.MessageContent, error) {
	for _, choice := range resp.Choices {
		for _, toolCall := range choice.ToolCalls {

			// Append tool_use to messageHistory
			assistantResponse := llms.MessageContent{
				Role: llms.ChatMessageTypeAI,
				Parts: []llms.ContentPart{
					llms.ToolCall{
						ID:   toolCall.ID,
						Type: toolCall.Type,
						FunctionCall: &llms.FunctionCall{
							Name:      toolCall.FunctionCall.Name,
							Arguments: toolCall.FunctionCall.Arguments,
						},
					},
				},
			}
			messageHistory = append(messageHistory, assistantResponse)

			var td ToolDef
			var found bool
			for _, tool := range tools {
				if tool.Name == toolCall.FunctionCall.Name {
					found = true
					td = tool
					break
				}
			}
			if !found {
				return messageHistory, errors.Errorf("Tool %s not found", toolCall.FunctionCall.Name)
			}

			callbackType := reflect.TypeOf(td.Callback)

			if callbackType.Kind() != reflect.Func || callbackType.NumIn() != 1 {
				return messageHistory, fmt.Errorf("callback function must have one parameter")
			}

			paramType := callbackType.In(0)
			v := reflect.New(paramType)

			err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), v.Interface())
			if err != nil {
				return messageHistory, errors.WithStack(err)
			}
			fmt.Printf("[tool %s %v]", td.Name, v.Elem().Interface())
			result := reflect.ValueOf(td.Callback).Call([]reflect.Value{v.Elem()})
			if len(result) > 0 {
				messageHistory = append(messageHistory, llms.MessageContent{
					Role: llms.ChatMessageTypeTool,
					Parts: []llms.ContentPart{
						llms.ToolCallResponse{
							ToolCallID: toolCall.ID,
							Name:       toolCall.FunctionCall.Name,
							Content:    result[0].String(),
						},
					},
				})
			}
		}
	}

	return messageHistory, nil
}
