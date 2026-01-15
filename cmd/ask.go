package cmd

import (
	"context"

	"github.com/elek/rai/config"
	"github.com/elek/rai/llm"
	"github.com/pkg/errors"
)

type Ask struct {
	llm.WithModel
	Message   string `arg:"" name:"message" help:"Message to send to Claude API"`
	WithTools bool   `help:"Enable all tools for the agent"`
}

func (a Ask) Run() error {
	ctx := context.Background()

	cfg, err := a.WithConfig.GetConfig()
	if err != nil {
		return errors.WithStack(err)
	}

	e := llm.NewExecutor(cfg)

	_, err = e.ExecPrompt(ctx, config.Model{}, "", a.Message, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
