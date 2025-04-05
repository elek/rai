package console

import (
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"unicode"
)

type KeyMap struct {
	CharacterForward        key.Binding
	CharacterBackward       key.Binding
	WordForward             key.Binding
	WordBackward            key.Binding
	DeleteWordBackward      key.Binding
	DeleteWordForward       key.Binding
	DeleteCharacterBackward key.Binding
	DeleteCharacterForward  key.Binding
	LineStart               key.Binding
	LineEnd                 key.Binding
}

var DefaultKeyMap = KeyMap{
	CharacterForward:        key.NewBinding(key.WithKeys("right", "ctrl+f")),
	CharacterBackward:       key.NewBinding(key.WithKeys("left", "ctrl+b")),
	DeleteCharacterBackward: key.NewBinding(key.WithKeys("backspace", "ctrl+h")),
	DeleteCharacterForward:  key.NewBinding(key.WithKeys("delete", "ctrl+d")),
	LineStart:               key.NewBinding(key.WithKeys("home", "ctrl+a")),
	LineEnd:                 key.NewBinding(key.WithKeys("end", "ctrl+e")),

	WordForward:        key.NewBinding(key.WithKeys("alt+right", "ctrl+right", "alt+f")),
	WordBackward:       key.NewBinding(key.WithKeys("alt+left", "ctrl+left", "alt+b")),
	DeleteWordBackward: key.NewBinding(key.WithKeys("alt+backspace", "ctrl+w")),
	DeleteWordForward:  key.NewBinding(key.WithKeys("alt+delete", "alt+d")),
}

type Textinput struct {
	Prompt string
	Text   string
	Width  int
	pos    int
	cursor cursor.Model
}

func NewTextInput() *Textinput {

	c := cursor.New()
	c.Focus()
	return &Textinput{
		Prompt: "> ",
		Text:   "",
		Width:  30,
		cursor: c,
	}
}

func (t *Textinput) Init() tea.Cmd {
	return t.cursor.SetMode(cursor.CursorBlink)
}

func (t *Textinput) Update(msg tea.Msg) (*Textinput, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.Width = msg.Width
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.DeleteCharacterBackward):
			tail, c, head := t.HeadAndTail()
			if tail != "" {
				t.Text = tail[:len(tail)-1] + c + head
				t.pos--
			}
		case key.Matches(msg, DefaultKeyMap.DeleteCharacterForward):
			tail, c, head := t.HeadAndTail()
			if c != "" {
				t.Text = tail + head
			}
		case key.Matches(msg, DefaultKeyMap.LineEnd):
			t.pos = len(t.Text)
		case key.Matches(msg, DefaultKeyMap.LineStart):
			t.pos = 0
		case key.Matches(msg, DefaultKeyMap.CharacterBackward):
			if t.pos > 0 {
				t.pos--
			}
		case key.Matches(msg, DefaultKeyMap.CharacterForward):
			if t.pos < len(t.Text) {
				t.pos++
			}
		case key.Matches(msg, DefaultKeyMap.WordBackward):
			initialPos := t.pos
			for {
				tail, _, _ := t.HeadAndTail()
				if tail == "" {
					break
				}
				if tail[len(tail)-1] == ' ' && t.pos != initialPos {
					break
				}
				t.pos--
			}
		case key.Matches(msg, DefaultKeyMap.WordForward):
			initialPos := t.pos
			for {
				tail, _, _ := t.HeadAndTail()
				if t.pos == len(t.Text) {
					break
				}
				if t.pos > 0 && tail[len(tail)-1] == ' ' && t.pos != initialPos {
					break
				}
				t.pos++
			}
		default:
			var filtered []rune
			for _, r := range msg.Runes {
				if unicode.IsPrint(r) {
					filtered = append(filtered, r)
				}
			}
			head, c, tail := t.HeadAndTail()
			if len(filtered) > 0 {
				t.Text = head + string(filtered) + c + tail
				t.pos += len(string(filtered))
			}
			return t, nil
		}
	}
	var cmds []tea.Cmd
	var cmd tea.Cmd

	t.cursor, cmd = t.cursor.Update(msg)
	cmds = append(cmds, cmd)
	return t, tea.Batch(cmds...)
}

func (t *Textinput) View() string {
	head, crs, tail := t.HeadAndTail()
	if crs == "" {
		crs = " "
	}
	t.cursor.SetChar(crs)
	//debug := fmt.Sprintf("t:%s pos:%d h:%s c:%s t:%s", t.Text, t.pos, head, crs, tail)
	debug := ""
	return ansi.Wrap(debug+t.Prompt+head+t.cursor.View()+tail, t.Width, "")

}

func (t *Textinput) Value() string {
	return t.Text
}

func (t *Textinput) SetValue(s string) {
	t.Text = s
	t.pos = len(s)
}

func (t *Textinput) HeadAndTail() (head string, cursor string, tail string) {
	if t.pos > 0 {
		if t.pos == len(t.Text) {
			head = t.Text
		} else {
			head = t.Text[:t.pos]
		}
	}

	if t.pos < len(t.Text) {
		cursor = t.Text[t.pos : t.pos+1]
		if t.pos < len(t.Text) {
			tail = t.Text[t.pos+1:]
		}

	}

	return head, cursor, tail
}
