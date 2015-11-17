package cmd

import (
	dcli "github.com/jeffjen/go-discovery/cli"

	cli "github.com/codegangsta/cli"
)

var (
	Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "addr",
			Usage: "API endpoint for admin",
		},
		cli.BoolFlag{
			Name:  "idle",
			Usage: "Set flag to disable active container life cycle event",
		},
		cli.BoolFlag{
			Name:  "persist",
			Usage: "Experimental: Set flag to persist data",
		},
	}
)

func NewFlag() []cli.Flag {
	return append(Flags, dcli.Flags...)
}
