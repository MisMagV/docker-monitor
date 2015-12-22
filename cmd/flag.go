package cmd

import (
	dcli "github.com/jeffjen/go-discovery/cli"
	scli "github.com/jeffjen/go-message/push/slack/cli"

	cli "github.com/codegangsta/cli"
)

var (
	Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "addr",
			Usage: "API endpoint for admin",
		},
		cli.StringFlag{
			Name:  "cluster",
			Usage: "cluster to apply for discovery",
			Value: "debug",
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

const (
	AgentReportTmpl = `{"endpoint": [[{{range $i, $k := .key}}{{if $i}},"{{$k}}"{{else}}"{{$k}}"{{end}}{{end}}]], "srv": "{{.Srv}}", "state": "{{.State}}"}`
)

func init() {
	scli.NotificationTmpl = AgentReportTmpl
}

func NewFlag() []cli.Flag {
	Flags = append(Flags, dcli.Flags...)
	Flags = append(Flags, scli.Flags...)
	return Flags
}
