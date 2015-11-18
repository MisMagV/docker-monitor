package main

import (
	"github.com/jeffjen/docker-monitor/cmd"

	cli "github.com/codegangsta/cli"

	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "docker-monitor"
	app.Usage = "Monitor docker events and report to discovery service"
	app.Authors = []cli.Author{
		cli.Author{"Yi-Hung Jen", "yihungjen@gmail.com"},
	}
	app.Flags = cmd.NewFlag()
	app.Commands = cmd.Commands
	app.Action = cmd.Monitor
	app.Run(os.Args)
}
