package llm

import (
	"context"
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/pkg/errors"
)

// defaultMaxTokens is used when a model config does not specify a token limit.
const defaultMaxTokens int64 = 4096

// anthropicModel implements Model against the official Anthropic SDK.
type anthropicModel struct {
	client    anthropic.Client
	model     string
	maxTokens int64
	debug     bool
}

// NewAnthropicModel creates a Model backed by the Anthropic Messages API. When
// debug is true, every request and response is traced to stderr.
func NewAnthropicModel(apiKey, baseURL, model string, maxTokens int64, debug bool) Model {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	return &anthropicModel{
		client:    anthropic.NewClient(opts...),
		model:     model,
		maxTokens: maxTokens,
		debug:     debug,
	}
}

func (m *anthropicModel) Provider() string { return "anthropic" }
func (m *anthropicModel) Name() string     { return m.model }

func (m *anthropicModel) Stream(ctx context.Context, req Request, onText func(delta string)) (*Turn, error) {
	params := anthropic.MessageNewParams{
		Model:     m.model,
		MaxTokens: m.maxTokens,
		Messages:  toAnthropicMessages(req.Messages),
	}
	if req.System != "" {
		params.System = []anthropic.TextBlockParam{{Text: req.System}}
	}
	if len(req.Tools) > 0 {
		params.Tools = toAnthropicTools(req.Tools)
	}
	// Temperature is only sent when explicitly set; the adaptive Opus models
	// reject the parameter entirely.
	if req.Temperature > 0 {
		params.Temperature = anthropic.Float(req.Temperature)
	}

	if m.debug {
		debugRequest(m.Provider(), m.model, req)
	}

	stream := m.client.Messages.NewStreaming(ctx, params)
	message := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		if err := message.Accumulate(event); err != nil {
			return nil, errors.WithStack(err)
		}
		if onText != nil {
			if delta, ok := event.AsAny().(anthropic.ContentBlockDeltaEvent); ok {
				if td, ok := delta.Delta.AsAny().(anthropic.TextDelta); ok {
					onText(td.Text)
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		return nil, errors.WithStack(err)
	}

	turn := turnFromAnthropic(&message)
	if m.debug {
		debugTurn(m.Provider(), m.model, turn)
	}
	return turn, nil
}

// toAnthropicMessages converts neutral messages into Anthropic message params.
// Tool results are carried in a user-role message, per the Anthropic API.
func toAnthropicMessages(msgs []Message) []anthropic.MessageParam {
	out := make([]anthropic.MessageParam, 0, len(msgs))
	for _, msg := range msgs {
		blocks := make([]anthropic.ContentBlockParamUnion, 0, len(msg.Blocks))
		for _, b := range msg.Blocks {
			switch b.Type {
			case BlockText:
				blocks = append(blocks, anthropic.NewTextBlock(b.Text))
			case BlockToolUse:
				input := json.RawMessage(b.Input)
				if len(input) == 0 {
					input = json.RawMessage("{}")
				}
				blocks = append(blocks, anthropic.NewToolUseBlock(b.ToolCallID, input, b.ToolName))
			case BlockToolResult:
				blocks = append(blocks, anthropic.NewToolResultBlock(b.ToolCallID, b.Text, b.IsError))
			}
		}
		switch msg.Role {
		case RoleAssistant:
			out = append(out, anthropic.NewAssistantMessage(blocks...))
		default: // user and tool results both go in a user-role message
			out = append(out, anthropic.NewUserMessage(blocks...))
		}
	}
	return out
}

// toAnthropicTools converts neutral tools into Anthropic tool params.
func toAnthropicTools(tools []Tool) []anthropic.ToolUnionParam {
	out := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		info := t.Info()
		tp := anthropic.ToolParam{
			Name: info.Name,
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: info.Parameters,
				Required:   info.Required,
			},
		}
		if info.Description != "" {
			tp.Description = anthropic.String(info.Description)
		}
		out = append(out, anthropic.ToolUnionParam{OfTool: &tp})
	}
	return out
}

// turnFromAnthropic converts an accumulated Anthropic message into a Turn.
func turnFromAnthropic(message *anthropic.Message) *Turn {
	var blocks []Block
	for _, block := range message.Content {
		switch v := block.AsAny().(type) {
		case anthropic.TextBlock:
			blocks = append(blocks, TextBlock(v.Text))
		case anthropic.ToolUseBlock:
			blocks = append(blocks, Block{
				Type:       BlockToolUse,
				ToolCallID: v.ID,
				ToolName:   v.Name,
				Input:      string(v.Input),
			})
		}
	}

	stop := StopEnd
	if message.StopReason == anthropic.StopReasonToolUse {
		stop = StopToolUse
	}

	return &Turn{
		Blocks: blocks,
		Usage: Usage{
			InputTokens:  message.Usage.InputTokens,
			OutputTokens: message.Usage.OutputTokens,
			TotalTokens:  message.Usage.InputTokens + message.Usage.OutputTokens,
		},
		StopReason: stop,
	}
}
