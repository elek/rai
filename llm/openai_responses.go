package llm

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
	"github.com/pkg/errors"
)

// openaiResponsesModel implements Model against the OpenAI Responses API. Unlike
// the Chat Completions implementation (openaiModel), it preserves the model's
// reasoning across tool-call turns by round-tripping opaque, provider-encrypted
// reasoning items. This is what reasoning models (o-series, gpt-5.x) need to
// stay coherent through a multi-turn tool loop.
type openaiResponsesModel struct {
	client    openai.Client
	model     string
	maxTokens int64
	debug     bool
}

// NewOpenAIResponsesModel creates a Model backed by the OpenAI Responses API. A
// non-empty baseURL targets an OpenAI-compatible endpoint (but note most
// OpenAI-compatible providers do not implement /v1/responses). When debug is
// true, every request and response is traced to stderr.
func NewOpenAIResponsesModel(apiKey, baseURL, model string, maxTokens int64, debug bool) Model {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	return &openaiResponsesModel{
		client:    openai.NewClient(opts...),
		model:     model,
		maxTokens: maxTokens,
		debug:     debug,
	}
}

func (m *openaiResponsesModel) Provider() string { return "openai-responses" }
func (m *openaiResponsesModel) Name() string     { return m.model }

func (m *openaiResponsesModel) Stream(ctx context.Context, req Request, onText func(delta string)) (*Turn, error) {
	params := responses.ResponseNewParams{
		Model: m.model,
		Input: responses.ResponseNewParamsInputUnion{OfInputItemList: toResponsesInput(req.System, req.Messages)},
		// Stay stateless: rather than let the server retain reasoning via a
		// stored response, we ask for the encrypted reasoning content and send
		// it back ourselves each turn. store:false without the include would
		// strip reasoning between turns, so the two settings go together.
		Store:   openai.Bool(false),
		Include: []responses.ResponseIncludable{responses.ResponseIncludableReasoningEncryptedContent},
	}
	if m.maxTokens > 0 {
		params.MaxOutputTokens = openai.Int(m.maxTokens)
	}
	if req.Temperature > 0 {
		params.Temperature = openai.Float(req.Temperature)
	}
	if len(req.Tools) > 0 {
		params.Tools = toResponsesTools(req.Tools)
	}

	if m.debug {
		debugRequest(m.Provider(), m.model, req)
	}

	stream := m.client.Responses.NewStreaming(ctx, params)
	var (
		items []responses.ResponseOutputItemUnion
		usage Usage
	)
	for stream.Next() {
		event := stream.Current()
		switch event.Type {
		case "response.output_text.delta":
			if onText != nil {
				onText(event.AsResponseOutputTextDelta().Delta)
			}
		case "response.output_item.done":
			items = append(items, event.AsResponseOutputItemDone().Item)
		case "response.completed":
			u := event.AsResponseCompleted().Response.Usage
			usage = Usage{InputTokens: u.InputTokens, OutputTokens: u.OutputTokens, TotalTokens: u.TotalTokens}
		case "error":
			e := event.AsError()
			return nil, errors.Errorf("responses stream error: %s", e.Message)
		}
	}
	if err := stream.Err(); err != nil {
		return nil, errors.WithStack(err)
	}

	blocks, stopReason := blocksFromResponseItems(items)
	turn := &Turn{Blocks: blocks, Usage: usage, StopReason: stopReason}
	if m.debug {
		debugTurn(m.Provider(), m.model, turn)
	}
	return turn, nil
}

// toResponsesInput converts the system prompt and neutral conversation into the
// Responses API input list. Order is preserved so each reasoning item precedes
// the function calls it produced, as the API requires.
func toResponsesInput(system string, msgs []Message) responses.ResponseInputParam {
	var input responses.ResponseInputParam
	if system != "" {
		input = append(input, responses.ResponseInputItemParamOfMessage(system, responses.EasyInputMessageRoleSystem))
	}
	for _, msg := range msgs {
		for _, b := range msg.Blocks {
			switch b.Type {
			case BlockText:
				role := responses.EasyInputMessageRoleUser
				if msg.Role == RoleAssistant {
					role = responses.EasyInputMessageRoleAssistant
				}
				input = append(input, responses.ResponseInputItemParamOfMessage(b.Text, role))
			case BlockReasoning:
				item := responses.ResponseInputItemParamOfReasoning(b.ToolCallID, []responses.ResponseReasoningItemSummaryParam{})
				item.OfReasoning.EncryptedContent = openai.String(b.Text)
				input = append(input, item)
			case BlockToolUse:
				args := b.Input
				if args == "" {
					args = "{}"
				}
				input = append(input, responses.ResponseInputItemParamOfFunctionCall(args, b.ToolCallID, b.ToolName))
			case BlockToolResult:
				input = append(input, responses.ResponseInputItemParamOfFunctionCallOutput(b.ToolCallID, b.Text))
			}
		}
	}
	return input
}

// toResponsesTools converts neutral tools into Responses API function tools.
func toResponsesTools(tools []Tool) []responses.ToolUnionParam {
	out := make([]responses.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		info := t.Info()
		// The Responses API validates the schema strictly: properties must be an
		// object and required must be an array, never null. A param-less tool has
		// nil maps/slices, so normalize them to empty values.
		properties := info.Parameters
		if properties == nil {
			properties = map[string]any{}
		}
		required := info.Required
		if required == nil {
			required = []string{}
		}
		params := map[string]any{
			"type":       "object",
			"properties": properties,
			"required":   required,
		}
		tool := responses.ToolParamOfFunction(info.Name, params, false)
		if info.Description != "" {
			tool.OfFunction.Description = openai.String(info.Description)
		}
		out = append(out, tool)
	}
	return out
}

// blocksFromResponseItems converts completed output items into neutral blocks,
// preserving order, and reports the stop reason (StopToolUse when any function
// call is present, otherwise StopEnd).
func blocksFromResponseItems(items []responses.ResponseOutputItemUnion) ([]Block, StopReason) {
	blocks := make([]Block, 0, len(items))
	stopReason := StopEnd
	for _, item := range items {
		switch item.Type {
		case "reasoning":
			r := item.AsReasoning()
			blocks = append(blocks, Block{Type: BlockReasoning, ToolCallID: r.ID, Text: r.EncryptedContent})
		case "function_call":
			fn := item.AsFunctionCall()
			blocks = append(blocks, Block{Type: BlockToolUse, ToolCallID: fn.CallID, ToolName: fn.Name, Input: fn.Arguments})
			stopReason = StopToolUse
		case "message":
			blocks = append(blocks, TextBlock(textOfResponseMessage(item.AsMessage())))
		}
	}
	return blocks, stopReason
}

// textOfResponseMessage concatenates the output_text parts of a message item.
func textOfResponseMessage(msg responses.ResponseOutputMessage) string {
	var s string
	for _, part := range msg.Content {
		s += part.AsOutputText().Text
	}
	return s
}
