package cmd

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/elek/rai/llm"
	"github.com/elek/rai/templates"
	"github.com/mattn/go-isatty"
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

	cfg, err := a.WithConfig.GetConfig()
	if err != nil {
		return errors.WithStack(err)
	}

	var cb llm.AgentCallback
	if a.DryRun {
		cb = llm.DryRun
	} else {

		e := llm.NewExecutor(cfg, a.Debug)
		cb = e.ExecPrompt
	}

	args := map[string]interface{}{
		"Args": a.Args,
	}

	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		stdinBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return errors.WithStack(err)
		}
		args["Stdin"] = string(stdinBytes)
	}
	_, err = templates.GoTemplateRender(cfg)(ctx, promptContent, args, cb)

	return errors.WithStack(err)
}
