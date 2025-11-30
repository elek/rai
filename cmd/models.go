package cmd

import (
	"context"
	"fmt"

	"github.com/elek/rai/util"
	"github.com/pkg/errors"
)

type Models struct {
	util.WithModel
}

func (l *Models) Run() error {
	ctx := context.Background()
	model, err := l.WithModel.CreateModel(ctx)
	if err != nil {
		return errors.WithStack(err)
	}
	if model.Lister == nil {
		fmt.Println("model list is not implemented")
		return nil
	}
	for _, name := range model.Lister() {
		fmt.Println(name)
	}

	return nil
}
