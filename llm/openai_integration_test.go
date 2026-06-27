package llm

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOpenAIIntegration hits the real OpenAI API when OPENAI_API_KEY is set, and
// is skipped otherwise. The model can be overridden via RAI_TEST_OPENAI_MODEL
// (default: gpt-4o-mini), and the base URL via OPENAI_BASE_URL.
func TestOpenAIIntegration(t *testing.T) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set; skipping live OpenAI integration test")
	}
	model := os.Getenv("RAI_TEST_OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	m := NewOpenAIModel(key, os.Getenv("OPENAI_BASE_URL"), model, 1024)
	agent := NewAgent(m, "", nil)
	res, err := agent.Run(context.Background(), "What is the capital of Spain? Answer in one word.", RunOptions{})
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(res.Text), "madrid")
}
