package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elek/rai/templates"
	"github.com/elek/rai/tool"
	"github.com/elek/rai/util"
	"github.com/pkg/errors"
	"github.com/tmc/langchaingo/llms"
)

type Do struct {
	util.WithModel
	Command string   `arg:"" name:"command" help:"Command to be executed"`
	Args    []string `arg:"" name:"args" help:"Arguments for the command" optional:""`
	DryRun  bool     `help:"Dry run (do not execute the command, just print the prompt)"`
}

func (a Do) Run() error {
	ctx := context.Background()
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.WithStack(err)
	}
	mpf := filepath.Join(home, ".config", "rai", a.Command)

	rawPrompt, err := os.ReadFile(mpf)
	if err != nil {
		return errors.WithStack(err)
	}

	rendered, err := templates.GoTemplateRender(string(rawPrompt), map[string]any{
		"Args": a.Args,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	if a.DryRun {
		fmt.Println("----- PROMPT -----")
		fmt.Println(rendered.Prompt)
		fmt.Println("----- END PROMPT -----")
		return nil
	}
	llm, err := a.CreateModel(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	var input []llms.MessageContent

	if rendered.System != "" {
		input = append(input, llms.TextParts(llms.ChatMessageTypeSystem, rendered.System))
	}

	input = append(input, llms.TextParts(llms.ChatMessageTypeHuman, rendered.Prompt))

	enabledTools := tool.AllTools()

	for {
		from := len(input)
		resp, err := llm.GenerateContent(
			ctx,
			input,
			llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				fmt.Print(string(chunk))
				return nil
			}),
			llms.WithTools(tool.AsFunction(enabledTools)),
		)
		if err != nil {
			return errors.WithStack(err)
		}
		fmt.Println()
		input, err = tool.HandleTools(ctx, llm, enabledTools, input, resp)
		if err != nil {
			return errors.WithStack(err)
		}
		if from == len(input) {
			break
		}
	}

	return errors.WithStack(err)
}
