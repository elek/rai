package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/elek/rai/acp"
	"github.com/elek/rai/config"
	"github.com/elek/rai/llm"
	"github.com/elek/rai/templates"
	"github.com/pkg/errors"
)

// Acp implements the `rai acp` CLI command which starts an ACP (Agent Client Protocol)
// server that communicates over stdio using JSON-RPC 2.0.
type Acp struct {
	llm.WithModel
	Command string `arg:"" name:"command" help:"Template name to configure the agent" optional:""`
}

// Run starts the ACP server, reading a template from the user's config directory
// and serving JSON-RPC messages over stdin/stdout.
func (a Acp) Run() error {
	ctx := context.Background()

	cfg, err := a.WithConfig.GetConfig()
	if err != nil {
		return errors.WithStack(err)
	}

	var parsed *templates.ParsedTemplate
	if a.Command != "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return errors.WithStack(err)
		}
		mpf := filepath.Join(home, ".config", "rai", a.Command)

		rawPrompt, err := os.ReadFile(mpf)
		if err != nil {
			return errors.WithStack(err)
		}

		parsed, err = templates.ParseTemplate(ctx, cfg, string(rawPrompt), nil)
		if err != nil {
			return errors.WithStack(err)
		}
		defer parsed.Close()
	}

	srv := acp.NewServer(parsed)
	srv.SetConfig(cfg)

	if a.WithModel.Model != "" {
		mod, found := cfg.FindModel(a.WithModel.Model)
		if !found {
			prov, modName, _ := strings.Cut(a.WithModel.Model, "/")
			mod = config.Model{
				Name:     a.WithModel.Model,
				Provider: prov,
				Model:    modName,
			}
		}
		srv.SetDefaultModel(mod)
	}

	return srv.Serve()
}
