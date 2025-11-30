package main

import (
	"log"

	"github.com/alecthomas/kong"
	kongyaml "github.com/alecthomas/kong-yaml"
	"github.com/elek/rai/cmd"
	"github.com/elek/rai/console"
)

type CLI struct {
	Ask    cmd.Ask     `cmd:"" help:"Ask a question to a model."`
	Do     cmd.Do      `cmd:"" help:"Run a command with custom prompts."`
	Run    console.Run `cmd:"" help:"Run an interactive conversation with a model."`
	Models cmd.Models  `cmd:"" help:"Models available models"`
}

func main() {
	var cli CLI
	ktx := kong.Parse(&cli, kong.Configuration(kongyaml.Loader, "~/.config/rai/config.yaml"))
	err := ktx.Run()
	if err != nil {
		log.Fatalf("%+v", err)
	}
}
