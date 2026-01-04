package llm

import (
	"context"
	"fmt"

	"charm.land/fantasy"
	"github.com/pkg/errors"
)

type AgentCallback func(ctx context.Context, model string, system string, prompt string, tools []fantasy.AgentTool) (string, error)

type Executor struct {
	model fantasy.LanguageModel
}

func NewExecutor(model fantasy.LanguageModel) *Executor {
	return &Executor{
		model: model,
	}
}

func (e *Executor) ExecPrompt(ctx context.Context, model string, system string, prompt string, tools []fantasy.AgentTool) (string, error) {
	var opts []fantasy.AgentOption

	opts = append(opts, fantasy.WithTools(tools...))

	if system != "" {
		opts = append(opts, fantasy.WithSystemPrompt(system))
	}

	agent := fantasy.NewAgent(
		e.model,
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

func DryRun(ctx context.Context, model string, system string, prompt string, tools []fantasy.AgentTool) (string, error) {
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
