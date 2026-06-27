package llm

import (
	"context"
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
// the complete turn. It never produces tool calls. Context cancellation is
// honored between words.
func (f *fakeModel) Stream(ctx context.Context, _ Request, onText func(delta string)) (*Turn, error) {
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
