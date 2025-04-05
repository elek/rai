package cmd

import (
	"context"
	"fmt"
	"github.com/elek/rai/providers"
	"github.com/elek/rai/schema"
	"github.com/pkg/errors"
	"os"
)

type Ask struct {
	providers.WithModel
	Message string `arg:"" name:"message" help:"Message to send to Claude API"`
}

func (a Ask) Run() error {
	ctx := context.Background()
	cv := &schema.Conversation{
		Messages: []schema.Message{
			{
				Role:    "user",
				Content: a.Message,
			},
		},
	}

	impl, model, err := a.CreateModel()
	if err != nil {
		return errors.WithStack(err)
	}

	msgs, usage, err := impl.Invoke(ctx, model, cv, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, msg := range msgs {
		fmt.Println(msg.Content)
	}
	_, _ = fmt.Fprint(os.Stderr, usage)
	return nil
}
