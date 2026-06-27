// Package llm defines a provider-neutral interface for large language models
// and an agent loop that drives multi-turn tool calling on top of it. Concrete
// implementations are provided for Anthropic, OpenAI, and a fake model used in
// tests.
package llm

// Usage reports token consumption for one or more model turns.
type Usage struct {
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
}

// Add returns the element-wise sum of two Usage values.
func (u Usage) Add(o Usage) Usage {
	return Usage{
		InputTokens:  u.InputTokens + o.InputTokens,
		OutputTokens: u.OutputTokens + o.OutputTokens,
		TotalTokens:  u.TotalTokens + o.TotalTokens,
	}
}

// Role identifies the author of a Message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// BlockType identifies the kind of content carried by a Block.
type BlockType string

const (
	// BlockText is plain text content.
	BlockText BlockType = "text"
	// BlockToolUse is a request from the assistant to call a tool.
	BlockToolUse BlockType = "tool_use"
	// BlockToolResult is the result of a tool call, sent back to the model.
	BlockToolResult BlockType = "tool_result"
)

// Block is a single piece of message content. Which fields are meaningful
// depends on Type:
//
//   - BlockText:       Text
//   - BlockToolUse:    ToolCallID, ToolName, Input
//   - BlockToolResult: ToolCallID, Text, IsError
type Block struct {
	Type       BlockType
	Text       string
	ToolCallID string
	ToolName   string
	Input      string
	IsError    bool
}

// Message is a single turn in a conversation.
type Message struct {
	Role   Role
	Blocks []Block
}

// TextBlock builds a plain text Block.
func TextBlock(text string) Block {
	return Block{Type: BlockText, Text: text}
}

// UserMessage builds a user message containing a single text block.
func UserMessage(text string) Message {
	return Message{Role: RoleUser, Blocks: []Block{TextBlock(text)}}
}
