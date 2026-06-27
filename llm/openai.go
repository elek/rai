package llm

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
	"github.com/pkg/errors"
)

// openaiModel implements Model against the official OpenAI SDK (chat completions).
type openaiModel struct {
	client    openai.Client
	model     string
	maxTokens int64
}

// NewOpenAIModel creates a Model backed by the OpenAI chat completions API. A
// non-empty baseURL targets an OpenAI-compatible endpoint.
func NewOpenAIModel(apiKey, baseURL, model string, maxTokens int64) Model {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	return &openaiModel{
		client:    openai.NewClient(opts...),
		model:     model,
		maxTokens: maxTokens,
	}
}

func (m *openaiModel) Provider() string { return "openai" }
func (m *openaiModel) Name() string     { return m.model }

func (m *openaiModel) Stream(ctx context.Context, req Request, onText func(delta string)) (*Turn, error) {
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(m.model),
		Messages: toOpenAIMessages(req.System, req.Messages),
		// Ask for usage to be reported on the final streamed chunk.
		StreamOptions: openai.ChatCompletionStreamOptionsParam{IncludeUsage: openai.Bool(true)},
	}
	if m.maxTokens > 0 {
		params.MaxCompletionTokens = openai.Int(m.maxTokens)
	}
	if req.Temperature > 0 {
		params.Temperature = openai.Float(req.Temperature)
	}
	if len(req.Tools) > 0 {
		params.Tools = toOpenAITools(req.Tools)
	}

	stream := m.client.Chat.Completions.NewStreaming(ctx, params)
	acc := openai.ChatCompletionAccumulator{}
	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)
		if onText != nil && len(chunk.Choices) > 0 {
			if d := chunk.Choices[0].Delta.Content; d != "" {
				onText(d)
			}
		}
	}
	if err := stream.Err(); err != nil {
		return nil, errors.WithStack(err)
	}

	return turnFromOpenAI(&acc), nil
}

// toOpenAIMessages converts neutral messages into OpenAI chat message params,
// prepending the system prompt when present.
func toOpenAIMessages(system string, msgs []Message) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(msgs)+1)
	if system != "" {
		out = append(out, openai.SystemMessage(system))
	}
	for _, msg := range msgs {
		switch msg.Role {
		case RoleAssistant:
			out = append(out, openaiAssistantMessage(msg.Blocks))
		case RoleTool:
			for _, b := range msg.Blocks {
				if b.Type == BlockToolResult {
					out = append(out, openai.ToolMessage(b.Text, b.ToolCallID))
				}
			}
		default: // user
			out = append(out, openai.UserMessage(textOf(msg.Blocks)))
		}
	}
	return out
}

// openaiAssistantMessage builds an assistant message param, including any
// tool_use blocks as OpenAI tool_calls.
func openaiAssistantMessage(blocks []Block) openai.ChatCompletionMessageParamUnion {
	var a openai.ChatCompletionAssistantMessageParam
	if text := textOf(blocks); text != "" {
		a.Content.OfString = openai.String(text)
	}
	for _, b := range blocks {
		if b.Type != BlockToolUse {
			continue
		}
		args := b.Input
		if args == "" {
			args = "{}"
		}
		a.ToolCalls = append(a.ToolCalls, openai.ChatCompletionMessageToolCallParam{
			ID: b.ToolCallID,
			Function: openai.ChatCompletionMessageToolCallFunctionParam{
				Name:      b.ToolName,
				Arguments: args,
			},
		})
	}
	return openai.ChatCompletionMessageParamUnion{OfAssistant: &a}
}

// toOpenAITools converts neutral tools into OpenAI function-tool params.
func toOpenAITools(tools []Tool) []openai.ChatCompletionToolParam {
	out := make([]openai.ChatCompletionToolParam, 0, len(tools))
	for _, t := range tools {
		info := t.Info()
		fn := shared.FunctionDefinitionParam{
			Name: info.Name,
			Parameters: shared.FunctionParameters{
				"type":       "object",
				"properties": info.Parameters,
				"required":   info.Required,
			},
		}
		if info.Description != "" {
			fn.Description = openai.String(info.Description)
		}
		out = append(out, openai.ChatCompletionToolParam{Function: fn})
	}
	return out
}

// turnFromOpenAI converts an accumulated chat completion into a Turn.
func turnFromOpenAI(acc *openai.ChatCompletionAccumulator) *Turn {
	turn := &Turn{StopReason: StopEnd}
	turn.Usage = Usage{
		InputTokens:  acc.Usage.PromptTokens,
		OutputTokens: acc.Usage.CompletionTokens,
		TotalTokens:  acc.Usage.TotalTokens,
	}

	if len(acc.Choices) == 0 {
		return turn
	}
	message := acc.Choices[0].Message
	if message.Content != "" {
		turn.Blocks = append(turn.Blocks, TextBlock(message.Content))
	}
	for _, tc := range message.ToolCalls {
		turn.Blocks = append(turn.Blocks, Block{
			Type:       BlockToolUse,
			ToolCallID: tc.ID,
			ToolName:   tc.Function.Name,
			Input:      tc.Function.Arguments,
		})
		turn.StopReason = StopToolUse
	}
	return turn
}
