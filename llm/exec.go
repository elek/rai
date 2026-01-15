package llm

import (
	"context"
	"fmt"

	"charm.land/fantasy"
	"github.com/elek/rai/config"
	"github.com/pkg/errors"
)

type AgentCallback func(ctx context.Context, model config.Model, system string, prompt string, tools []fantasy.AgentTool) (string, error)

type Executor struct {
	cfg config.Config
}

func NewExecutor(cfg config.Config) *Executor {
	return &Executor{
		cfg: cfg,
	}
}

func (e *Executor) ExecPrompt(ctx context.Context, mdl config.Model, system string, prompt string, tools []fantasy.AgentTool) (string, error) {
	var opts []fantasy.AgentOption

	opts = append(opts, fantasy.WithTools(tools...))

	if system != "" {
		opts = append(opts, fantasy.WithSystemPrompt(system))
	}

	mc := mdl
	if mc == (config.Model{}) {
		var found bool
		mc, found = e.cfg.FindDefaultModel()
		if !found {
			return "", errors.New("no default model configured")
		}
	}
	model, err := create(ctx, e.cfg, mc)
	if err != nil {
		return "", errors.WithStack(err)
	}

	agent := fantasy.NewAgent(
		model,
		opts...,
	)

	response, err := agent.Stream(ctx, fantasy.AgentStreamCall{
		Prompt: prompt,
		OnTextDelta: func(id string, token string) error {
			fmt.Print(token)
			return nil
		},
		OnTextEnd: func(id string) error {
			fmt.Println()
			return nil
		},
		OnToolCall: func(toolCall fantasy.ToolCallContent) error {
			fmt.Println("Calling tool", toolCall.ToolName, "with input:", toolCall.Input)
			return nil
		},
	})
	if err != nil {
		return "", errors.WithStack(err)
	}

	return response.Response.Content.Text(), nil
}

func DryRun(ctx context.Context, model config.Model, system string, prompt string, tools []fantasy.AgentTool) (string, error) {
	fmt.Println("Using model:", model.Provider, model.Model)
	if system != "" {
		fmt.Println("--- SYSTEM PROMPT ---")
		fmt.Println(system)
	}
	fmt.Println("--- PROMPT ----------")
	fmt.Println(prompt)
	fmt.Println("--- TOOLS -----------")
	for _, tool := range tools {
		fmt.Println("   *", tool.Info().Name)
	}
	fmt.Println("---------------------")

	return "", nil
}
