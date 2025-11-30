package util

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicModels returns a function that lists all available Anthropic Claude models via API
func AnthropicModels(key string) func() []string {
	return func() []string {
		client := anthropic.NewClient(
			option.WithAPIKey(key),
		)

		ctx := context.Background()
		models, err := client.Models.List(ctx, anthropic.ModelListParams{})
		if err != nil {
			return []string{}
		}

		var modelIDs []string
		for _, model := range models.Data {
			modelIDs = append(modelIDs, model.ID)
		}

		return modelIDs
	}
}
