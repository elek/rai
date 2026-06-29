package llm

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"strings"
)

// fakeModel is a Model that returns randomly generated text instead of calling
// a real provider. It is intended for tests and offline experiments (e.g.
// exercising the ACP server end-to-end without API keys). It never emits tool
// calls.
type fakeModel struct {
	provider string
	model    string
}

// NewFakeModel creates a fake model bound to the given provider and model names.
// The names are only echoed through Provider()/Name(); they have no effect on
// the generated output.
func NewFakeModel(provider, model string) Model {
	return &fakeModel{provider: provider, model: model}
}

func (f *fakeModel) Provider() string { return f.provider }
func (f *fakeModel) Name() string     { return f.model }

// Stream streams randomly generated text word by word via onText, then returns
// the complete turn. As a special case, when the prompt mentions "commit" it
// requests git tool calls instead (see commitTurn). Context cancellation is
// honored between words.
func (f *fakeModel) Stream(ctx context.Context, req Request, onText func(delta string)) (*Turn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if turn := commitTurn(req); turn != nil {
		return turn, nil
	}

	text := randomText()

	for _, word := range strings.SplitAfter(text, " ") {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if word == "" {
			continue
		}
		if onText != nil {
			onText(word)
		}
	}

	return &Turn{
		Blocks:     []Block{TextBlock(text)},
		Usage:      usageFor(text),
		StopReason: StopEnd,
	}, nil
}

// commitTurn implements the fake model's only tool-using behavior: when the
// prompt mentions "commit", it asks the agent to stage all changes and commit
// them with a random message. It returns nil to fall through to plain text
// generation when the prompt is unrelated. Once the agent has fed the tool
// results back (the last message is a tool turn), it returns a plain
// completion so the agent loop terminates instead of committing forever.
func commitTurn(req Request) *Turn {
	if n := len(req.Messages); n > 0 && req.Messages[n-1].Role == RoleTool {
		text := "Done: staged and committed the changes."
		return &Turn{Blocks: []Block{TextBlock(text)}, Usage: usageFor(text), StopReason: StopEnd}
	}
	if !mentionsCommit(req.Messages) {
		return nil
	}
	message := randomCommitMessage()
	return &Turn{
		Blocks: []Block{
			gitToolUse("fake-tool-1", "git add -A"),
			gitToolUse("fake-tool-2", `git commit -m "`+message+`"`),
		},
		Usage:      usageFor(message),
		StopReason: StopToolUse,
	}
}

// mentionsCommit reports whether any user message text contains "commit".
func mentionsCommit(messages []Message) bool {
	for _, m := range messages {
		if m.Role != RoleUser {
			continue
		}
		for _, b := range m.Blocks {
			if b.Type == BlockText && strings.Contains(strings.ToLower(b.Text), "commit") {
				return true
			}
		}
	}
	return false
}

// gitToolUse builds a tool_use block invoking the "git" tool with the given
// shell command.
func gitToolUse(id, command string) Block {
	input, _ := json.Marshal(map[string]string{"command": command})
	return Block{Type: BlockToolUse, ToolCallID: id, ToolName: "git", Input: string(input)}
}

// randomCommitMessage returns a short commit message of 2-5 random words.
func randomCommitMessage() string {
	n := 2 + rand.IntN(4)
	words := make([]string, n)
	for i := range words {
		words[i] = randomWord()
	}
	return strings.Join(words, " ")
}

// usageFor builds a plausible Usage value for the given generated text.
func usageFor(text string) Usage {
	out := int64(len(strings.Fields(text)))
	in := int64(8)
	return Usage{InputTokens: in, OutputTokens: out, TotalTokens: in + out}
}

// randomWord returns a random lowercase word between 3 and 10 characters long.
func randomWord() string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	n := 3 + rand.IntN(8)
	var b strings.Builder
	b.Grow(n)
	for i := 0; i < n; i++ {
		b.WriteByte(letters[rand.IntN(len(letters))])
	}
	return b.String()
}

// randomParagraph returns a paragraph of random words.
func randomParagraph() string {
	n := 15 + rand.IntN(36)
	words := make([]string, n)
	for i := range words {
		words[i] = randomWord()
	}
	return strings.Join(words, " ")
}

// randomText returns 1-3 paragraphs of random words separated by blank lines.
func randomText() string {
	n := 1 + rand.IntN(3)
	paragraphs := make([]string, n)
	for i := range paragraphs {
		paragraphs[i] = randomParagraph()
	}
	return strings.Join(paragraphs, "\n\n")
}
