package config

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type WithConfig struct {
	ConfigFile string
	cached     *Config
}

func (w *WithConfig) GetConfig() (Config, error) {
	if w.cached != nil {
		return *w.cached, nil
	}
	if w.ConfigFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Config{}, errors.WithStack(err)
		}
		w.ConfigFile = filepath.Join(home, ".config", "rai", "config.yaml")
	}
	file, err := os.ReadFile(w.ConfigFile)
	if err != nil {
		return Config{}, errors.WithStack(err)
	}
	var cfg Config
	err = yaml.Unmarshal(file, &cfg)
	if err != nil {
		return Config{}, errors.WithStack(err)
	}
	for name, provider := range cfg.Providers {
		if provider.Type == "" {
			provider.Type = provider.Name
		}
		if provider.Name == "" {
			provider.Name = provider.Type
		}
		cfg.Providers[name] = provider
	}
	for name, model := range cfg.Models {
		if model.MaxToken == 0 {
			model.MaxToken = 1024
		}
		if model.Temperature == 0 {
			model.Temperature = 0.7
		}
		cfg.Models[name] = model
	}
	w.cached = &cfg
	return cfg, nil
}
