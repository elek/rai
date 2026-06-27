package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeModelStreamsReassembleToFullText(t *testing.T) {
	m := NewFakeModel("fake", "fake-1")
	assert.Equal(t, "fake", m.Provider())
	assert.Equal(t, "fake-1", m.Name())

	var streamed strings.Builder
	turn, err := m.Stream(context.Background(), Request{}, func(d string) {
		streamed.WriteString(d)
	})
	require.NoError(t, err)
	require.Len(t, turn.Blocks, 1)

	full := turn.Blocks[0].Text
	assert.Equal(t, full, streamed.String(), "streamed deltas should reassemble to the full text")
	assert.Equal(t, StopEnd, turn.StopReason)
	assert.Positive(t, turn.Usage.OutputTokens)
}

func TestFakeModelHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewFakeModel("fake", "fake-1").Stream(ctx, Request{}, nil)
	require.Error(t, err)
}

func TestFakeModelDrivesAgent(t *testing.T) {
	agent := NewAgent(NewFakeModel("fake", "fake-1"), "", nil)
	res, err := agent.Run(context.Background(), "hello", RunOptions{})
	require.NoError(t, err)
	assert.NotEmpty(t, res.Text)
}
