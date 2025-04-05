package console

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/elek/rai/providers"
	"github.com/elek/rai/schema"
)

type Run struct {
	providers.WithModel
}

func (a Run) Run() error {

	c := &schema.Conversation{}

	impl, model, err := a.CreateModel()
	if err != nil {
		return err
	}
	ix := 0

	var history []string
	for {
		input := NewInput(fmt.Sprintf("%s(%d)", model.Model, len(c.Messages)), history)
		input.AddAction(tea.KeyCtrlL, func() tea.Cmd {
			c.Messages = []schema.Message{}
			input.textinput.Text = ""
			ix = 0
			return tea.Quit
		})
		app := tea.NewProgram(input)
		_, err := app.Run()
		if err != nil {
			return err
		}

		if input.stop {
			return nil
		}
		query := input.textinput.Text
		if query == "exit" {
			return nil
		}
		if query == "" {
			continue
		}
		history = append(history, query)
		c.Messages = append(c.Messages, schema.Message{
			Role:    "user",
			Content: query,
		})
		ix = printMessages(c.Messages, ix)

		ctx := context.Background()

		messages, _, err := impl.Invoke(ctx, model, c, nil)
		if err != nil {
			fmt.Println(err.Error())
			messages = append(messages, schema.Message{
				Role:    "error",
				Content: err.Error(),
			})

		}

		for _, msg := range messages {
			c.Messages = append(c.Messages, msg)
		}
		ix = printMessages(c.Messages, ix)

	}

}

func printMessages(messages []schema.Message, ix int) int {
	fmt.Println(ix, len(messages))
	for ix < len(messages) {
		fmt.Println(messages[ix].Role + ": " + messages[ix].Content)
		ix++
	}
	return ix
}
