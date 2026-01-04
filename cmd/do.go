package cmd

import (
	"context"
	"os"
	"path/filepath"

	"github.com/elek/rai/llm"
	"github.com/elek/rai/templates"
	"github.com/pkg/errors"
)

type Do struct {
	llm.WithModel
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

	promptContent := string(rawPrompt)

	var cb llm.AgentCallback
	if a.DryRun {
		cb = llm.DryRun
	} else {
		model, err := a.CreateModel(ctx)
		if err != nil {
			return errors.WithStack(err)
		}

		e := llm.NewExecutor(model)
		cb = e.ExecPrompt
	}

	args := map[string]interface{}{
		"Args": a.Args,
	}
	_, err = templates.GoTemplateRender(ctx, promptContent, args, cb)

	return errors.WithStack(err)
}
