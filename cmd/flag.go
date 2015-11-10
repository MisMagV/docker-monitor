package cmd

import (
	cli "github.com/codegangsta/cli"
)

var (
	Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "advertise",
			Usage: "The netloc of this node seen by other nodes",
		},
		cli.StringFlag{
			Name:  "heartbeat",
			Value: "60s",
			Usage: "Rate at which monitor will announce alive",
		},
		cli.StringFlag{
			Name:  "ttl",
			Value: "90s",
			Usage: "Expire time for which montior is considered offline",
		},
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
