package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/elek/rai/providers"
	"github.com/elek/rai/schema"
	"github.com/elek/rai/tool"
	"github.com/pkg/errors"
	"os/exec"
	"strings"
)

//go:embed system.txt
var systemPrompt string

//go:embed review.txt
var reviewPrompt string

type Review struct {
	providers.WithModel
}

// GerritReview represents the review data to be sent to GerritEnabled

type ReviewOutput struct {
	File    string
	Line    string
	Comment string
}

func (a Review) Run() error {
	ctx := context.Background()

	cmd := exec.Command("git", "show", "HEAD")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.WithStack(err)
	}

	impl, model, err := a.CreateModel()
	if err != nil {
		return errors.WithStack(err)
	}

	pr := strings.ReplaceAll(reviewPrompt, "{{.patch}}", string(out))

	cv := &schema.Conversation{
		System: commitPrompt,
		Messages: []schema.Message{
			{
				Role:    "user",
				Content: pr,
			},
		},
	}

	tools := tool.AllTools()

	for {
		msgs, _, err := impl.Invoke(ctx, model, cv, tools)
		if err != nil {
			return errors.WithStack(err)
		}
		for _, m := range msgs {
			fmt.Println(m.Content, m.ToolName, m.ToolID)
		}
		msgs, handled, err := tool.HandleTools(msgs, tools)
		if err != nil {
			return errors.WithStack(err)
		}
		if !handled {
			break
		}
		cv.Append(msgs)

	}
	return nil
}
