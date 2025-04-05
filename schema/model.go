package schema

import (
	"context"
	"github.com/elek/rai/config"
	"reflect"
	"strconv"
)

type Conversation struct {
	System   string
	Messages []Message
}

func (c *Conversation) Append(msgs []Message) {
	c.Messages = append(c.Messages, msgs...)
}

type Tool struct {
	Name        string
	Description string
	Callback    any
	Params      []Param
}

type Param struct {
	Name        string
	Type        reflect.Type
	Description string
}

type Message struct {
	Role     string
	Content  string
	ToolName string
	ToolID   string
	Original any
}

type ModelVersion struct {
	ID   string
	Name string
}

// Usage represents token usage information for a model invocation
type Usage struct {
	InputTokens  int
	OutputTokens int
}

func (u Usage) String() string {
	return "Input tokens: " + strconv.Itoa(u.InputTokens) + ", Output tokens: " + strconv.Itoa(u.OutputTokens)
}

func (u Usage) Add(other Usage) Usage {
	return Usage{
		InputTokens:  u.InputTokens + other.InputTokens,
		OutputTokens: u.OutputTokens + other.OutputTokens,
	}
}

type Model interface {
	Invoke(ctx context.Context, model config.Model, c *Conversation, tools []Tool) ([]Message, Usage, error)
	ListModels(ctx context.Context) ([]ModelVersion, error)
}
