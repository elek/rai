package util

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoogleModels(t *testing.T) {
	// Skip test if GOOGLE_API_KEY is not set
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		t.Skip("GOOGLE_API_KEY not set, skipping test")
	}

	lister := GoogleModels(apiKey)
	models := lister()

	// Verify we got at least some models back
	assert.NotEmpty(t, models, "Expected at least one model to be returned")

	// Verify all model IDs are non-empty strings
	for _, model := range models {
		assert.NotEmpty(t, model, "Model ID should not be empty")
	}

	t.Logf("Found %d models: %v", len(models), models)
}
