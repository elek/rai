package main

import (
	"github.com/alecthomas/kong"
	"github.com/alecthomas/kong-yaml"
	"github.com/elek/rai/cmd"
	"github.com/elek/rai/console"
	"github.com/elek/rai/gui"
	"log"
)

type CLI struct {
	List   cmd.List    `cmd:"" help:"List available models."`
	Ask    cmd.Ask     `cmd:"" help:"Ask a question to a model."`
	Run    console.Run `cmd:"" help:"Run a conversation with a model."`
	Review cmd.Review  `cmd:"" help:"Review a patch."`
	Commit cmd.Commit  `cmd:"" help:"Commit a model."`
	Do     cmd.Do      `cmd:"" help:"Run a command with custom prompts."`
	Gui    gui.Gui     `cmd:"" help:"Run the GUI."`
}

func main() {
	var cli CLI
	ktx := kong.Parse(&cli, kong.Configuration(kongyaml.Loader, "~/.config/rai/config.yaml"))
	err := ktx.Run()
	if err != nil {
		log.Fatalf("%+v", err)
	}
}
