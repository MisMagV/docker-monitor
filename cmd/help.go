package cmd

import (
	cli "github.com/codegangsta/cli"
)

func init() {
	cli.AppHelpTemplate = `Usage: {{.Name}} {{if .Flags}}[OPTIONS]{{end}} DISCOVERY_URI

{{.Usage}}

Version: {{.Version}}

DISCOVERY_URI:
	EXAMPLE URI - etcd://10.0.1.10:2379,10.0.1.11:2379

{{if .Flags}}Options:
	{{range .Flags}}{{.}}
	{{end}}{{end}}
`
}
