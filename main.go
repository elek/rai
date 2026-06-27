package main

import (
	"log"

	"github.com/alecthomas/kong"
	kongyaml "github.com/alecthomas/kong-yaml"
	"github.com/elek/rai/cmd"
)

type CLI struct {
	Ask    cmd.Ask    `cmd:"" help:"Ask a question to a model."`
	Do     cmd.Do     `cmd:"" help:"Run a command with custom prompts."`
	Models cmd.Models `cmd:"" help:"Models available models"`
	Acp    cmd.Acp    `cmd:"" help:"Start ACP (Agent Client Protocol) server."`
}

func main() {
	var cli CLI
	ktx := kong.Parse(&cli, kong.Configuration(kongyaml.Loader, "~/.config/rai/config.yaml"))
	err := ktx.Run()
	if err != nil {
		log.Fatalf("%++v", err)
	}
}
