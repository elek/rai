package gui

import (
	"context"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/elek/rai/providers"
	"github.com/elek/rai/schema"
	"image/color"
)

type Gui struct {
	providers.WithModel
}

type biggerTextTheme struct{}

func (b biggerTextTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, variant)
}

func (b biggerTextTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (b biggerTextTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (b biggerTextTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 24
	case theme.SizeNameCaptionText:
		return 14
	case theme.SizeNameSubHeadingText:
		return 20
	case theme.SizeNameHeadingText:
		return 24
	default:
		return theme.DefaultTheme().Size(name)
	}
}

func (g Gui) Run() error {
	a := app.New()
	a.Settings().SetTheme(&biggerTextTheme{})
	w := a.NewWindow("RAI - LLM Chat")
	w.Resize(fyne.NewSize(800, 600))

	chatDisplay := widget.NewRichText()
	chatDisplay.Wrapping = fyne.TextWrapWord

	scrollContainer := container.NewScroll(chatDisplay)
	scrollContainer.SetMinSize(fyne.NewSize(800, 500))

	c := &schema.Conversation{}

	inputField := NewMyEntry(func() {
		c.Messages = []schema.Message{}
		chatDisplay.Segments = []widget.RichTextSegment{}
		chatDisplay.Refresh()
	})
	inputField.SetPlaceHolder("Type your message here...")
	inputField.MultiLine = true

	impl, model, err := g.CreateModel()
	if err != nil {
		return err
	}

	ctx := context.Background()

	submitMessage := func(text string) {
		if text != "" {

			c.Messages = append(c.Messages, schema.Message{
				Role:    "user",
				Content: text,
			})

			chatDisplay.AppendMarkdown(displayMessages(c.Messages))
			scrollContainer.ScrollToBottom()

			go func() {
				messages, _, err := impl.Invoke(ctx, model, c, nil)
				if err != nil {
					messages = append(messages, schema.Message{
						Role:    "error",
						Content: err.Error(),
					})

				}

				for _, msg := range messages {
					c.Messages = append(c.Messages, msg)
				}
				chatDisplay.ParseMarkdown(displayMessages(c.Messages))
				scrollContainer.ScrollToBottom()
			}()

		}
	}

	inputField.OnSubmitted = func(text string) {
		submitMessage(text)
		inputField.SetText("")
	}

	ctrlL := &desktop.CustomShortcut{
		KeyName:  fyne.KeyL,
		Modifier: desktop.ControlModifier,
	}
	w.Canvas().AddShortcut(ctrlL, func(shortcut fyne.Shortcut) {
		c.Messages = []schema.Message{}
		inputField.SetText("")
	})

	content := container.NewBorder(
		nil,

		inputField,

		nil,
		nil,
		scrollContainer,
	)

	w.SetContent(content)
	w.Canvas().Focus(inputField)

	w.ShowAndRun()
	return nil
}

func displayMessages(messages []schema.Message) string {
	out := ""
	for _, m := range messages {
		out += "" + m.Role + ": " + m.Content + "\n"
	}
	return out
}

type myEntry struct {
	widget.Entry
	reset func()
}

func NewMyEntry(reset func()) *myEntry {
	entry := &myEntry{
		reset: reset,
	}
	entry.SetPlaceHolder("Type your message here...")
	entry.MultiLine = true
	entry.ExtendBaseWidget(entry)
	return entry
}

func (e *myEntry) TypedShortcut(shortcut fyne.Shortcut) {
	if sc, ok := shortcut.(*desktop.CustomShortcut); ok {
		if sc.KeyName == fyne.KeyL && sc.Modifier == desktop.ControlModifier {
			e.SetText("")
			e.reset()
			return
		}
	}
	e.Entry.TypedShortcut(shortcut)
}
