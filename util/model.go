package util

import (
	"context"
	"strings"

	"github.com/elek/rai/config"
	"github.com/pkg/errors"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/openai"
)

type WithModel struct {
	config.WithConfig
	Model string `help:"model to be used" default:""`
	//MaxToken    int     `help:"maximum number of tokens to be used" default:"1024"`
	//Temperature float64 `help:"temperature to be used" default:"0.7"`
	//Model       string  `help:"model to be used" default:""`
	//MaxToken    int     `help:"maximum number of tokens to be used" default:"1024"`
	//Temperature float64 `help:"temperature to be used" default:"0.7"`
	Provider string `help:"force to use the given provider (anthropic, openrouter, openai, google)"`
	Debug    bool   `help:"enable debug mode"`
}

type Model struct {
	Name string
	llms.Model
	Lister func() []string
}

func (w WithModel) CreateModel(ctx context.Context) (Model, error) {
	cfg, err := w.WithConfig.GetConfig()
	if err != nil {
		return Model{}, errors.New("config couldn't be read")
	}

	create := func(model config.Model) (Model, error) {
		p, found := cfg.FindProvider(model.Provider)
		if !found {
			return Model{}, errors.New("provider couldn't be found: " + model.Provider)
		}

		switch model.Provider {
		case "anthropic":
			ar, err := anthropic.New(anthropic.WithModel(model.Model), anthropic.WithToken(p.Key))
			return Model{
				Name:   model.Name,
				Model:  ar,
				Lister: AnthropicModels(p.Key),
			}, errors.WithStack(err)
		case "gemini", "google":
			ar, err := googleai.New(ctx, googleai.WithDefaultModel(model.Model), googleai.WithAPIKey(p.Key))
			return Model{
				Name:   model.Name,
				Model:  ar,
				Lister: GoogleModels(p.Key),
			}, errors.WithStack(err)
		case "openrouter":
			ar, err := openai.New(openai.WithModel(model.Model), openai.WithToken(p.Key), openai.WithBaseURL("https://openrouter.ai/api/v1"))
			return Model{
				Name:  model.Name,
				Model: ar,
			}, errors.WithStack(err)
		case "openai":
			ar, err := openai.New(openai.WithModel(model.Model), openai.WithToken(p.Key))
			return Model{
				Name:  model.Name,
				Model: ar,
			}, errors.WithStack(err)
		}
		return Model{}, errors.New("unknown provider: " + model.Provider)
	}

	if w.Model == "" {
		mod, found := cfg.FindDefaultModel()
		if !found {
			return Model{}, errors.New("model is not defined, and no default model found")
		}
		return create(mod)
	}
	mod, found := cfg.FindModel(w.Model)
	if !found {
		prov, mod, _ := strings.Cut(w.Model, "/")
		return create(config.Model{
			Name:     w.Model,
			Provider: prov,
			Model:    mod,
		})
	}
	return create(mod)
}
