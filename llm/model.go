package llm

import (
	"context"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
	"charm.land/fantasy/providers/google"
	"charm.land/fantasy/providers/openrouter"
	"github.com/elek/rai/config"
	"github.com/pkg/errors"
	"github.com/tmc/langchaingo/llms"
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

func create(ctx context.Context, cfg config.Config, model config.Model) (fantasy.LanguageModel, error) {
	p, found := cfg.FindProvider(model.Provider)
	if !found {
		return nil, errors.New("provider couldn't be found: " + model.Provider)
	}

	switch model.Provider {
	case "anthropic":
		provider, err := anthropic.New(anthropic.WithAPIKey(p.Key))
		if err != nil {
			return nil, errors.WithStack(err)
		}

		model, err := provider.LanguageModel(ctx, model.Model)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return model, nil
	case "openrouter":
		provider, err := openrouter.New(openrouter.WithAPIKey(p.Key))
		if err != nil {
			return nil, errors.WithStack(err)
		}

		model, err := provider.LanguageModel(ctx, model.Model)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return model, nil
	case "google":
		var ops []google.Option
		if p.Project != "" && p.Location != "" {
			ops = append(ops, google.WithVertex(p.Project, p.Location))
		} else if p.Key != "" {
			ops = append(ops, google.WithGeminiAPIKey(p.Key))
		}

		provider, err := google.New(ops...)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		model, err := provider.LanguageModel(ctx, model.Model)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return model, nil
	default:
		fmt.Println(p)
		return nil, errors.New("unknown provider: " + model.Provider)
	}
}

func (w WithModel) CreateModel(ctx context.Context) (fantasy.LanguageModel, error) {
	cfg, err := w.WithConfig.GetConfig()
	if err != nil {
		return nil, errors.New("config couldn't be read")
	}

	if w.Model == "" {
		mod, found := cfg.FindDefaultModel()
		if !found {
			return nil, errors.New("model is not defined, and no default model found")
		}
		return create(ctx, cfg, mod)
	}
	mod, found := cfg.FindModel(w.Model)
	if !found {
		prov, mod, _ := strings.Cut(w.Model, "/")
		return create(ctx, cfg, config.Model{
			Name:     w.Model,
			Provider: prov,
			Model:    mod,
		})
	}
	return create(ctx, cfg, mod)
}
