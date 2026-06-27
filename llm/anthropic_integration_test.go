package llm

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnthropicIntegration hits the real Anthropic API when ANTHROPIC_API_KEY is
// set, and is skipped otherwise. The model can be overridden via
// RAI_TEST_ANTHROPIC_MODEL (default: claude-haiku-4-5).
func TestAnthropicIntegration(t *testing.T) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		t.Skip("ANTHROPIC_API_KEY not set; skipping live Anthropic integration test")
	}
	model := os.Getenv("RAI_TEST_ANTHROPIC_MODEL")
	if model == "" {
		model = "claude-haiku-4-5"
	}

	m := NewAnthropicModel(key, "", model, 1024, false)
	agent := NewAgent(m, "", nil)
	res, err := agent.Run(context.Background(), "What is the capital of Spain? Answer in one word.", RunOptions{})
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(res.Text), "madrid")
}
