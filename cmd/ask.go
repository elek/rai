package cmd

import (
	"context"
	"fmt"

	"github.com/elek/rai/util"
	"github.com/pkg/errors"
	"github.com/tmc/langchaingo/llms"
)

type Ask struct {
	util.WithModel
	Message string `arg:"" name:"message" help:"Message to send to Claude API"`
}

func (a Ask) Run() error {
	ctx := context.Background()
	llm, err := a.CreateModel(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	input := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, a.Message),
	}

	_, err = llm.GenerateContent(
		ctx,
		input,
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			fmt.Print(string(chunk))
			return nil
		}),
	)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
