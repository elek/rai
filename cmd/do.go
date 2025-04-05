package cmd

import (
	"bytes"
	"context"
	"fmt"
	"github.com/elek/rai/providers"
	"github.com/elek/rai/schema"
	"github.com/pkg/errors"
	"html/template"
	"os"
	"path/filepath"
)

type Do struct {
	providers.WithModel
	Command string `arg:"" name:"command" help:"Command to be executed"`
	File    string `help:"Additional file which can be used in prompt template"`
}

func (a Do) Run() error {
	ctx := context.Background()
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.WithStack(err)
	}
	spf := filepath.Join(home, ".config", "rai", a.Command+".system")
	mpf := filepath.Join(home, ".config", "rai", a.Command)

	rawPrompt, err := os.ReadFile(mpf)
	if err != nil {
		return errors.WithStack(err)
	}
	prompt := string(rawPrompt)

	fileContent := ""
	if a.File != "" {
		raw, err := os.ReadFile(a.File)
		if err != nil {
			return errors.WithStack(err)
		}
		fileContent = string(raw)
	}

	tpl, err := template.New("prompt").Parse(string(prompt))
	if err != nil {
		return errors.WithStack(err)
	}
	out := bytes.NewBuffer([]byte{})
	err = tpl.Execute(out, map[string]string{
		"file": fileContent,
	})
	prompt = out.String()

	cv := &schema.Conversation{
		Messages: []schema.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	if _, err := os.Stat(spf); err == nil {
		sprompt, err := os.ReadFile(spf)
		if err != nil {
			return errors.WithStack(err)
		}
		cv.System = string(sprompt)
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
