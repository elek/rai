package cmd

import (
	"context"
	"testing"

	"github.com/elek/rai/config"
	"github.com/elek/rai/llm"
	"github.com/stretchr/testify/assert"
)

func TestModelOverrideUsesCLIModelWhenSet(t *testing.T) {
	cli := config.Model{Name: "claude", Provider: "anthropic", Model: "claude-sonnet"}
	template := config.Model{Name: "openai", Provider: "openai", Model: "gpt-5.5"}

	var got config.Model
	base := func(_ context.Context, m config.Model, _ string, _ string, _ []llm.Tool) (string, error) {
		got = m
		return "", nil
	}

	_, err := withModelOverride(base, cli)(context.Background(), template, "", "", nil)
	assert.NoError(t, err)
	assert.Equal(t, cli, got, "the CLI model must override the template model")
}

func TestModelOverrideKeepsTemplateModelWhenCLIEmpty(t *testing.T) {
	template := config.Model{Name: "openai", Provider: "openai", Model: "gpt-5.5"}

	var got config.Model
	base := func(_ context.Context, m config.Model, _ string, _ string, _ []llm.Tool) (string, error) {
		got = m
		return "", nil
	}

	_, err := withModelOverride(base, config.Model{})(context.Background(), template, "", "", nil)
	assert.NoError(t, err)
	assert.Equal(t, template, got, "with no CLI model the template's model must be used")
}
