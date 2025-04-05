package console

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Input struct {
	textinput  *Textinput
	stop       bool
	history    []string
	historyIdx int
	actions    map[tea.KeyType]func() tea.Cmd
}

func NewInput(prompt string, history []string) *Input {
	model := NewTextInput()
	model.Prompt = prompt + "> "
	model.Width = 30
	return &Input{
		textinput:  model,
		history:    history,
		historyIdx: -1,
		actions:    make(map[tea.KeyType]func() tea.Cmd),
	}
}

func (i *Input) AddAction(key tea.KeyType, action func() tea.Cmd) {
	i.actions[key] = action
}

func (i *Input) Init() tea.Cmd {
	return textinput.Blink
}

func (i *Input) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if action, found := i.actions[msg.Type]; found {
			return i, action()
		}
		switch msg.Type {
		case tea.KeyEnter:
			return i, tea.Quit
		case tea.KeyCtrlQ:
			i.stop = true
			return i, tea.Quit
		case tea.KeyCtrlC:
			if i.textinput.Value() == "" {
				i.stop = true
				return i, tea.Quit
			} else {
				i.textinput.SetValue("")
				return i, nil
			}
		case tea.KeyEsc:
			i.stop = true
			return i, tea.Quit
		case tea.KeyUp:
			if i.historyIdx == -1 {
				i.historyIdx = len(i.history)
			}
			if i.historyIdx > 0 {
				i.historyIdx--
			}
			if len(i.history) > 0 {
				i.textinput.SetValue(i.history[i.historyIdx])
			}
		case tea.KeyDown:
			switch {
			case i.historyIdx == -1:
			case i.historyIdx < len(i.history)-1:
				i.historyIdx++
				i.textinput.SetValue(i.history[i.historyIdx])
			case i.historyIdx == len(i.history)-1:
				i.historyIdx = -1
				i.textinput.SetValue("")
			}
		default:
		}
	}

	var cmd tea.Cmd
	i.textinput, cmd = i.textinput.Update(msg)
	return i, cmd
}

func (i *Input) View() string {
	return i.textinput.View() + "\n"
}

var _ tea.Model = &Input{}
