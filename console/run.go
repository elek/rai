package console

import (
	"github.com/elek/rai/llm"
)

type Run struct {
	llm.WithModel
}

func (a Run) Run() error {
	//
	//var messageHistory []llms.MessageContent
	//
	//ctx := context.Background()
	//
	//llm, err := a.CreateModel(ctx)
	//if err != nil {
	//	return errors.WithStack(err)
	//}
	//
	//var history []string
	//for {
	//	input := NewInput(fmt.Sprintf("%s(%d)", llm.Name, len(messageHistory)), history)
	//	input.AddAction(tea.KeyCtrlL, func() tea.Cmd {
	//		messageHistory = messageHistory[:0]
	//		input.textinput.Text = ""
	//		return tea.Quit
	//	})
	//	app := tea.NewProgram(input)
	//	_, err := app.Run()
	//	if err != nil {
	//		return err
	//	}
	//
	//	if input.stop {
	//		return nil
	//	}
	//	query := input.textinput.Text
	//	if query == "exit" {
	//		return nil
	//	}
	//	if query == "" {
	//		continue
	//	}
	//	history = append(history, query)
	//	messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeHuman, query))
	//	fmt.Println()
	//	resp, err := llm.GenerateContent(
	//		ctx,
	//		messageHistory,
	//		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
	//			fmt.Print(string(chunk))
	//			return nil
	//		}),
	//	)
	//	if err != nil {
	//		return errors.WithStack(err)
	//	}
	//
	//	for _, choice := range resp.Choices {
	//		if choice.Content != "" {
	//			messageHistory = append(messageHistory, llms.TextParts(llms.ChatMessageTypeAI, choice.Content))
	//		}
	//	}
	return nil
	//}

}
