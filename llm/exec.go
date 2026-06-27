package llm

import (
	"context"
	"fmt"

	"github.com/elek/rai/config"
	"github.com/pkg/errors"
)

// AgentCallback runs a prompt against a model with the given system prompt and
// tools, returning the model's final text response. ExecPrompt and DryRun both
// satisfy this type.
type AgentCallback func(ctx context.Context, model config.Model, system string, prompt string, tools []Tool) (string, error)

// Executor runs prompts against models created from a configuration.
type Executor struct {
	cfg config.Config
}

// NewExecutor creates an Executor bound to a configuration.
func NewExecutor(cfg config.Config) *Executor {
	return &Executor{cfg: cfg}
}

// ExecPrompt runs prompt through an agent loop, streaming the model's text to
// stdout and reporting tool calls as they happen. If mdl is the zero value, the
// configured default model is used.
func (e *Executor) ExecPrompt(ctx context.Context, mdl config.Model, system string, prompt string, tools []Tool) (string, error) {
	if mdl == (config.Model{}) {
		def, found := e.cfg.FindDefaultModel()
		if !found {
			return "", errors.New("no default model configured")
		}
		mdl = def
	}

	model, err := NewModel(ctx, e.cfg, mdl)
	if err != nil {
		return "", errors.WithStack(err)
	}

	agent := NewAgent(model, system, tools)
	result, err := agent.Run(ctx, prompt, RunOptions{
		OnTextDelta: func(delta string) { fmt.Print(delta) },
		OnToolCall: func(name, input string) {
			fmt.Println("Calling tool", name, "with input:", input)
		},
	})
	if err != nil {
		return "", errors.WithStack(err)
	}
	fmt.Println()
	return result.Text, nil
}

// DryRun prints the resolved model, prompts, and tools without calling any
// provider. It satisfies AgentCallback.
func DryRun(ctx context.Context, model config.Model, system string, prompt string, tools []Tool) (string, error) {
	fmt.Println("Using model:", model.Provider, model.Model)
	if system != "" {
		fmt.Println("--- SYSTEM PROMPT ---")
		fmt.Println(system)
	}
	fmt.Println("--- PROMPT ----------")
	fmt.Println(prompt)
	fmt.Println("--- TOOLS -----------")
	fmt.Print(describeTools(tools))
	fmt.Println("---------------------")
	return "", nil
}
