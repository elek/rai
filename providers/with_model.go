package providers

import (
	"github.com/elek/rai/config"
	"github.com/elek/rai/schema"
	"github.com/pkg/errors"
	"strings"
)

type Implementation func(provider config.Provider) schema.Model

var Implementations = map[string]Implementation{
	"anthropic": func(provider config.Provider) schema.Model {
		return NewClaudeModel(provider)
	},
	"openrouter": func(provider config.Provider) schema.Model {
		return NewOpenRouter(provider)
	},
	"google": func(provider config.Provider) schema.Model {
		return NewGeminiModel(provider)
	},
	"openai": func(provider config.Provider) schema.Model {
		return NewOpenAIModel(provider)
	},
}

type WithModel struct {
	config.WithConfig
	Model       string  `help:"model to be used" default:""`
	MaxToken    int     `help:"maximum number of tokens to be used" default:"1024"`
	Temperature float64 `help:"temperature to be used" default:"0.7"`
	Provider    string  `help:"force to use the given provider (anthropic, openrouter, openai, google)"`
	Debug       bool    `help:"enable debug mode"`
}

func (w WithModel) CreateModel() (schema.Model, config.Model, error) {
	cfg, err := w.WithConfig.GetConfig()
	if err != nil {
		return nil, config.Model{}, errors.New("config couldn't be read")
	}

	if w.Model == "" {
		model, found := cfg.FindDefaultModel()
		if !found {
			return nil, config.Model{}, errors.New("model is not defined, and no default model found")
		}
		provider, found := cfg.FindProvider(model.Provider)
		if !found {
			return nil, config.Model{}, errors.New("model is not defined, and no default model found")
		}
		return Implementations[provider.Type](provider), model, nil
	}
	model, found := cfg.FindModel(w.Model)
	if !found {
		prov, mod, _ := strings.Cut(w.Model, "/")
		model := config.Model{
			Name:        mod,
			Model:       mod,
			MaxToken:    w.MaxToken,
			Temperature: w.Temperature,
			Provider:    prov,
			Debug:       w.Debug,
		}
		provider, found := cfg.FindProvider(model.Provider)
		if !found {
			return nil, config.Model{}, errors.New("model is not defined, and no default model found")
		}
		return Implementations[provider.Type](provider), model, nil
	}
	provider, found := cfg.FindProvider(model.Provider)
	if !found {
		return nil, config.Model{}, errors.New("model is not defined, and no default model found")
	}
	return Implementations[provider.Type](provider), model, nil
}
