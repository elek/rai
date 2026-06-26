package llm

import (
	"context"
	"math/rand/v2"
	"strings"

	"charm.land/fantasy"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// fakeModel is a fantasy.LanguageModel implementation that returns randomly
// generated text instead of calling a real provider. It is intended for tests
// and offline experiments (e.g. exercising the ACP server end-to-end without
// API keys). It never emits tool calls.
type fakeModel struct {
	provider string
	model    string
}

// newFakeModel creates a fake language model bound to the given provider and
// model names. The names are only echoed back through Provider()/Model();
// they have no effect on the generated output.
func newFakeModel(provider, model string) fantasy.LanguageModel {
	return &fakeModel{provider: provider, model: model}
}

// Provider returns the configured provider name.
func (f *fakeModel) Provider() string { return f.provider }

// Model returns the configured model name.
func (f *fakeModel) Model() string { return f.model }

// randomWord returns a random lowercase word between 3 and 10 characters long.
func randomWord() string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	n := 3 + rand.IntN(8) // 3..10 inclusive
	var b strings.Builder
	b.Grow(n)
	for i := 0; i < n; i++ {
		b.WriteByte(letters[rand.IntN(len(letters))])
	}
	return b.String()
}

// randomParagraph returns a paragraph of random words.
func randomParagraph() string {
	n := 15 + rand.IntN(36) // 15..50 words
	words := make([]string, n)
	for i := range words {
		words[i] = randomWord()
	}
	return strings.Join(words, " ")
}

// randomText returns 1-3 paragraphs of random words separated by blank lines.
func randomText() string {
	n := 1 + rand.IntN(3) // 1..3 paragraphs
	paragraphs := make([]string, n)
	for i := range paragraphs {
		paragraphs[i] = randomParagraph()
	}
	return strings.Join(paragraphs, "\n\n")
}

// usageFor builds a plausible Usage struct for the given generated text.
func usageFor(text string) fantasy.Usage {
	out := int64(len(strings.Fields(text)))
	in := int64(8)
	return fantasy.Usage{
		InputTokens:  in,
		OutputTokens: out,
		TotalTokens:  in + out,
	}
}

// Generate returns a single response containing randomly generated text.
func (f *fakeModel) Generate(_ context.Context, _ fantasy.Call) (*fantasy.Response, error) {
	text := randomText()
	return &fantasy.Response{
		Content:      fantasy.ResponseContent{fantasy.TextContent{Text: text}},
		FinishReason: fantasy.FinishReasonStop,
		Usage:        usageFor(text),
	}, nil
}

// Stream streams randomly generated text word by word. It emits a text block
// (start, per-word deltas, end) followed by a finish part. No tool calls are
// ever produced. Context cancellation is honored between deltas.
func (f *fakeModel) Stream(ctx context.Context, _ fantasy.Call) (fantasy.StreamResponse, error) {
	text := randomText()
	id := uuid.New().String()

	return func(yield func(fantasy.StreamPart) bool) {
		if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeTextStart, ID: id}) {
			return
		}

		// Stream the text in whitespace-preserving chunks so the reassembled
		// text matches what Generate would produce.
		for _, word := range strings.SplitAfter(text, " ") {
			if err := ctx.Err(); err != nil {
				yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeError, ID: id, Error: err})
				return
			}
			if word == "" {
				continue
			}
			if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeTextDelta, ID: id, Delta: word}) {
				return
			}
		}

		if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeTextEnd, ID: id}) {
			return
		}

		yield(fantasy.StreamPart{
			Type:         fantasy.StreamPartTypeFinish,
			ID:           id,
			Usage:        usageFor(text),
			FinishReason: fantasy.FinishReasonStop,
		})
	}, nil
}

// GenerateObject is not supported by the fake model.
func (f *fakeModel) GenerateObject(_ context.Context, _ fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	return nil, errors.New("fake model does not support structured object generation")
}

// StreamObject is not supported by the fake model.
func (f *fakeModel) StreamObject(_ context.Context, _ fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	return nil, errors.New("fake model does not support structured object generation")
}
