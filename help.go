package main

import (
	cli "github.com/codegangsta/cli"
)

func init() {
	cli.AppHelpTemplate = `Usage: {{.Name}} {{if .Flags}}[OPTIONS]{{end}} DISCOVERY_URI

{{.Usage}}

Version: {{.Version}}{{if or .Author .Email}}

Author:{{if .Author}}
	{{.Author}}{{if .Email}} - <{{.Email}}>{{end}}{{else}}
    {{.Email}}{{end}}{{end}}

DISCOVERY_URI:
	EXAMPLE URI - etcd://10.0.1.10:2379,10.0.1.11:2379{{if .Flags}}

Options:
	{{range .Flags}}{{.}}
	{{end}}{{end}}
`
}
