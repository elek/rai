package cmd

import (
	"context"
	"fmt"
	"github.com/elek/rai/providers"
	"github.com/pkg/errors"
)

type List struct {
	providers.WithModel
}

func (l List) Run() error {
	ctx := context.Background()
	cfg, err := l.GetConfig()
	if err != nil {
		return errors.WithStack(err)
	}
	for _, provider := range cfg.Providers {
		prov, found := providers.Implementations[provider.Type]
		if !found {
			return errors.Errorf("provider %s not found", provider.Type)
		}
		impl := prov(provider)
		resp, err := impl.ListModels(ctx)
		if err != nil {
			return errors.WithStack(err)
		}
		for _, r := range resp {
			fmt.Println(r.ID, r.Name)
		}
	}

	return nil
}
