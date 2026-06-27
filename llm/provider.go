package llm

import (
	"context"
	"strings"

	"github.com/elek/rai/config"
	"github.com/pkg/errors"
)

// WithModel is a Kong CLI mixin that resolves which model to use for a command.
type WithModel struct {
	config.WithConfig
	Model    string `help:"model to be used" default:""`
	Provider string `help:"force to use the given provider (anthropic, openai, fake)"`
	Debug    bool   `help:"enable debug mode"`
}

// NewModel creates a Model from the given config and model definition. Supported
// provider types in this phase are: anthropic, openai (and openai-compatible),
// and fake.
func NewModel(ctx context.Context, cfg config.Config, model config.Model) (Model, error) {
	p, found := cfg.FindProvider(model.Provider)
	if !found {
		return nil, errors.New("provider couldn't be found: " + model.Provider)
	}

	maxTokens := int64(model.MaxToken)
	switch p.Type {
	case "fake":
		return NewFakeModel(model.Provider, model.Model), nil
	case "anthropic":
		return NewAnthropicModel(p.Key, p.Endpoint, model.Model, maxTokens, model.Debug), nil
	case "openai", "openaicompat":
		return NewOpenAIModel(p.Key, p.Endpoint, model.Model, maxTokens, model.Debug), nil
	case "google", "openrouter":
		return nil, errors.New("provider type not supported in this phase: " + p.Type)
	default:
		return nil, errors.New("unknown provider type: " + p.Type)
	}
}

// ResolveModel resolves the --model flag to a config.Model. If no model is
// specified it returns an empty config.Model (the caller falls back to the
// configured default). A "provider/model" value that is not in the config is
// accepted as an ad-hoc model definition.
func (w WithModel) ResolveModel(cfg config.Config) (config.Model, error) {
	if w.Model == "" {
		return config.Model{}, nil
	}
	if mod, found := cfg.FindModel(w.Model); found {
		return mod, nil
	}
	prov, modName, ok := strings.Cut(w.Model, "/")
	if !ok {
		return config.Model{}, errors.Errorf("model %q not found in config and not in provider/model format", w.Model)
	}
	return config.Model{Name: w.Model, Provider: prov, Model: modName}, nil
}

// CreateModel resolves the configured model and instantiates it.
func (w WithModel) CreateModel(ctx context.Context) (Model, error) {
	cfg, err := w.GetConfig()
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
		mdl = mod
	}
	// The --debug flag forces tracing on regardless of the model config.
	if w.Debug {
		mdl.Debug = true
	}
	return NewModel(ctx, cfg, mdl)
}
