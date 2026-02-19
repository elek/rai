package llm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
	"charm.land/fantasy/providers/google"
	"charm.land/fantasy/providers/openaicompat"
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
	Provider string `help:"force to use the given provider (anthropic, openrouter, openai, openaicompat, google)"`
	Debug    bool   `help:"enable debug mode"`
}

type Model struct {
	Name string
	llms.Model
	Lister func() []string
}

// NewLanguageModel creates a fantasy.LanguageModel from the given config and model definition.
func NewLanguageModel(ctx context.Context, cfg config.Config, model config.Model) (fantasy.LanguageModel, error) {
	p, found := cfg.FindProvider(model.Provider)
	if !found {
		return nil, errors.New("provider couldn't be found: " + model.Provider)
	}

	switch p.Type {
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
		if p.CredentialFile != "" {
			if err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", p.CredentialFile); err != nil {
				return nil, errors.WithStack(err)
			}
		}

		var ops []google.Option
		if p.Project != "" && p.Location != "" {
			ops = append(ops, google.WithVertex(p.Location, p.Project))
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
	case "openai", "openaicompat":
		ops := []openaicompat.Option{
			openaicompat.WithAPIKey(p.Key),
		}
		if p.Endpoint != "" {
			ops = append(ops, openaicompat.WithBaseURL(p.Endpoint))
		}
		provider, err := openaicompat.New(ops...)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		lm, err := provider.LanguageModel(ctx, model.Model)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return lm, nil
	default:
		fmt.Println(p)
		return nil, errors.New("unknown provider: " + model.Provider)
	}
}

// ResolveModel resolves the --model flag to a config.Model.
// If no model is specified, it returns an empty config.Model (caller should fall back to default).
func (w WithModel) ResolveModel(cfg config.Config) (config.Model, error) {
	if w.Model == "" {
		return config.Model{}, nil
	}
	mod, found := cfg.FindModel(w.Model)
	if found {
		return mod, nil
	}
	prov, modName, ok := strings.Cut(w.Model, "/")
	if !ok {
		return config.Model{}, errors.Errorf("model %q not found in config and not in provider/model format", w.Model)
	}
	return config.Model{
		Name:     w.Model,
		Provider: prov,
		Model:    modName,
	}, nil
}

func (w WithModel) CreateModel(ctx context.Context) (fantasy.LanguageModel, error) {
	cfg, err := w.WithConfig.GetConfig()
	if err != nil {
		return nil, errors.New("config couldn't be read")
	}

	mdl, err := w.ResolveModel(cfg)
	if err != nil {
		return nil, err
	}
	if mdl == (config.Model{}) {
		mod, found := cfg.FindDefaultModel()
		if !found {
			return nil, errors.New("model is not defined, and no default model found")
		}
		return NewLanguageModel(ctx, cfg, mod)
	}
	return NewLanguageModel(ctx, cfg, mdl)
}
