package util

import (
	"context"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GoogleModels returns a function that lists all available Google Generative AI models via API
func GoogleModels(key string) func() []string {
	return func() []string {
		ctx := context.Background()
		client, err := genai.NewClient(ctx, option.WithAPIKey(key))
		if err != nil {
			return []string{}
		}
		defer client.Close()

		iter := client.ListModels(ctx)
		var modelIDs []string
		for {
			model, err := iter.Next()
			if err != nil {
				break
			}
			modelIDs = append(modelIDs, model.Name)
		}

		return modelIDs
	}
}
