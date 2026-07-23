package cmd

import (
	"context"
	"os"
	"path/filepath"

	"github.com/elek/rai/config"
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

	cfg, err := a.GetConfig()
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

	// A --model/--provider flag on the CLI takes precedence over the template's
	// <model> tag; when unset, the template's choice (or the default) is used.
	cliModel, err := a.ResolveModel(cfg)
	if err != nil {
		return errors.WithStack(err)
	}
	cb = withModelOverride(cb, cliModel)

	args := map[string]interface{}{
		"Args": a.Args,
	}

	//if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
	//	stdinBytes, err := io.ReadAll(os.Stdin)
	//	if err != nil {
	//		return errors.WithStack(err)
	//	}
	//	args["Stdin"] = string(stdinBytes)
	//}
	_, err = templates.GoTemplateRender(cfg)(ctx, promptContent, args, cb)

	return errors.WithStack(err)
}

// withModelOverride wraps an AgentCallback so that a non-empty model forces the
// model used for the call, overriding whatever the template resolved. When
// model is the zero value the template's model (passed by the renderer) is left
// untouched.
func withModelOverride(base llm.AgentCallback, model config.Model) llm.AgentCallback {
	return func(ctx context.Context, templateModel config.Model, system string, prompt string, tools []llm.Tool) (string, error) {
		if model != (config.Model{}) {
			templateModel = model
		}
		return base(ctx, templateModel, system, prompt, tools)
	}
}
