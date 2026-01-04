package cmd

import (
	"github.com/elek/catwalk-open/pkg/providers"
)

type Models struct {
	Provider string `arg:"" optional:"" help:"The provider to list models for."`
}

func (l *Models) Run() error {
	for _, provider := range providers.GetAll() {
		if l.Provider != "" && provider.Name != l.Provider {
			continue
		}
		for _, model := range provider.Models {
			println(string(provider.Type) + ": " + model.ID)
		}
	}
	return nil
}
