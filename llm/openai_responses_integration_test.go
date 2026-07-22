package llm

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOpenAIResponsesIntegration hits the real OpenAI Responses API when
// OPENAI_API_KEY is set, and is skipped otherwise. It drives a multi-turn tool
// loop against a reasoning model, which is the scenario that degenerates on the
// Chat Completions path — so it proves the encrypted-reasoning round-trip keeps
// the model coherent enough to call the tool and return a final answer.
//
// The model can be overridden via RAI_TEST_RESPONSES_MODEL (default: gpt-5.5)
// and the base URL via OPENAI_BASE_URL.
func TestOpenAIResponsesIntegration(t *testing.T) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set; skipping live OpenAI Responses integration test")
	}
	model := os.Getenv("RAI_TEST_RESPONSES_MODEL")
	if model == "" {
		model = "gpt-5.5"
	}

	type secretIn struct{}
	var called bool
	secret := NewTool[secretIn]("get_secret_number", "returns the secret number", func(_ context.Context, _ secretIn) (string, error) {
		called = true
		return "the secret number is four hundred and seven", nil
	})

	m := NewOpenAIResponsesModel(key, os.Getenv("OPENAI_BASE_URL"), model, 4096, false)
	agent := NewAgent(m, "You are a helpful assistant. Use tools when needed.", []Tool{secret})
	res, err := agent.Run(context.Background(),
		"Call the get_secret_number tool, then tell me the secret number.",
		RunOptions{})
	require.NoError(t, err)
	assert.True(t, called, "the model should have called the tool")
	require.NotEmpty(t, strings.TrimSpace(res.Text), "the model must return a final answer after the tool call")
	text := strings.ToLower(res.Text)
	assert.True(t, strings.Contains(text, "407") || strings.Contains(text, "four hundred and seven"),
		"final answer should report the secret number, got: %q", res.Text)
}
