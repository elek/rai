package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/elek/rai/providers"
	"github.com/elek/rai/schema"
	"github.com/elek/rai/tool"
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"strings"
)

//go:embed commit.txt
var commitPrompt string

type Commit struct {
	providers.WithModel
}

func (c Commit) Run() error {
	ctx := context.Background()

	err := exec.Command("git", "add", ".").Run()
	if err != nil {
		return errors.WithStack(err)
	}

	patch, err := exec.Command("git", "diff", "--cached").CombinedOutput()
	if err != nil {
		return errors.WithStack(err)
	}

	files, err := exec.Command("git", "diff", "--cached", "--name-only").CombinedOutput()
	if err != nil {
		return errors.WithStack(err)
	}

	fileContents := ""
	for _, file := range strings.Split(string(files), "\n") {
		if file == "" {
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		fileContents += "Full content of the file " + file + " is:\n\n```\n" + string(content) + "\n```\n\n"
	}

	impl, model, err := c.CreateModel()
	if err != nil {
		return errors.WithStack(err)
	}

	pr := strings.ReplaceAll(commitPrompt, "{{.patch}}", string(patch))
	pr = strings.ReplaceAll(pr, "{{.files}}", fileContents)

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

	allUsage := schema.Usage{}
	for {
		msgs, usage, err := impl.Invoke(ctx, model, cv, tools)
		if err != nil {
			return errors.WithStack(err)
		}
		allUsage = allUsage.Add(usage)
		msgs, handled, err := tool.HandleTools(msgs, tools)
		if err != nil {
			return errors.WithStack(err)
		}
		if !handled {
			break
		}
		cv.Append(msgs)
	}
	fmt.Println(allUsage)
	return nil

}
