package main

import (
	dcli "github.com/jeffjen/go-discovery/cli"
	node "github.com/jeffjen/go-discovery/info"
	scli "github.com/jeffjen/go-message/push/slack/cli"

	cli "github.com/codegangsta/cli"

	"path"
	tmpl "text/template"
	"time"
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

func init() {
	scli.NotificationTmpl = `{"timestamp": "{{now}}", "node": {{here}}, "endpoint": [{{range $i, $k := .Key}}{{if $i}},"{{base $k}}"{{else}}"{{base $k}}"{{end}}{{end}}], "srv": "{{.Srv}}", "state": "{{.State}}"}`

	scli.FuncMap = tmpl.FuncMap{
		"base": path.Base,
		"here": func() string { return node.MetaData },
		"now":  func() string { return time.Now().String() },
	}
}

func NewFlag() []cli.Flag {
	Flags = append(Flags, dcli.Flags...)
	Flags = append(Flags, scli.Flags...)
	return Flags
}
