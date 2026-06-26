package llm

import (
	"context"
	"strings"
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeModel must satisfy the fantasy.LanguageModel interface.
var _ fantasy.LanguageModel = (*fakeModel)(nil)

// assertRandomText checks that text is 1-3 paragraphs of words each 3-10 chars.
func assertRandomText(t *testing.T, text string) {
	t.Helper()
	require.NotEmpty(t, text)

	paragraphs := strings.Split(text, "\n\n")
	assert.GreaterOrEqual(t, len(paragraphs), 1)
	assert.LessOrEqual(t, len(paragraphs), 3)

	for _, p := range paragraphs {
		words := strings.Fields(p)
		assert.NotEmpty(t, words)
		for _, w := range words {
			assert.GreaterOrEqual(t, len(w), 3, "word %q too short", w)
			assert.LessOrEqual(t, len(w), 10, "word %q too long", w)
		}
	}
}

func TestFakeModelGenerate(t *testing.T) {
	m := newFakeModel("fake", "fake-model")
	assert.Equal(t, "fake", m.Provider())
	assert.Equal(t, "fake-model", m.Model())

	resp, err := m.Generate(context.Background(), fantasy.Call{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, fantasy.FinishReasonStop, resp.FinishReason)
	assertRandomText(t, resp.Content.Text())
	assert.Positive(t, resp.Usage.OutputTokens)
}

func TestFakeModelStream(t *testing.T) {
	m := newFakeModel("fake", "fake-model")

	stream, err := m.Stream(context.Background(), fantasy.Call{})
	require.NoError(t, err)

	var (
		sawStart, sawEnd, sawFinish bool
		assembled                   strings.Builder
		finishReason                fantasy.FinishReason
	)
	for part := range stream {
		switch part.Type {
		case fantasy.StreamPartTypeTextStart:
			sawStart = true
		case fantasy.StreamPartTypeTextDelta:
			assembled.WriteString(part.Delta)
		case fantasy.StreamPartTypeTextEnd:
			sawEnd = true
		case fantasy.StreamPartTypeFinish:
			sawFinish = true
			finishReason = part.FinishReason
		case fantasy.StreamPartTypeError:
			t.Fatalf("unexpected error part: %v", part.Error)
		}
	}

	assert.True(t, sawStart, "expected text start part")
	assert.True(t, sawEnd, "expected text end part")
	assert.True(t, sawFinish, "expected finish part")
	assert.Equal(t, fantasy.FinishReasonStop, finishReason)
	assertRandomText(t, assembled.String())
}

func TestFakeModelStreamCancelled(t *testing.T) {
	m := newFakeModel("fake", "fake-model")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before consuming

	stream, err := m.Stream(ctx, fantasy.Call{})
	require.NoError(t, err)

	var sawError bool
	for part := range stream {
		if part.Type == fantasy.StreamPartTypeError {
			sawError = true
			assert.ErrorIs(t, part.Error, context.Canceled)
		}
	}
	assert.True(t, sawError, "expected an error part when context is cancelled")
}
