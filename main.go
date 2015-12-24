package main

import (
	dkr "github.com/jeffjen/docker-monitor/docker"
	up "github.com/jeffjen/docker-monitor/upkeep"
	web "github.com/jeffjen/docker-monitor/web"
	disc "github.com/jeffjen/go-discovery"
	dcli "github.com/jeffjen/go-discovery/cli"
	scli "github.com/jeffjen/go-message/push/slack/cli"

	log "github.com/Sirupsen/logrus"
	cli "github.com/codegangsta/cli"

	"os"
	"path"
)

const (
	DiscoveryPath = "/docker/swarm/nodes"
)

func main() {
	app := cli.NewApp()
	app.Name = "docker-monitor"
	app.Usage = "Monitor docker events and report to discovery service"
	app.Authors = []cli.Author{
		cli.Author{"Yi-Hung Jen", "yihungjen@gmail.com"},
	}
	app.Flags = NewFlag()
	app.Action = Monitor
	app.Run(os.Args)
}

func Monitor(ctx *cli.Context) {
	var (
		addr = ctx.String("addr")
		idle = ctx.Bool("idle")

		stop = make(chan struct{}, 1)
	)

	// setup register path for discovery
	disc.RegisterPath = path.Join(ctx.String("cluster"), DiscoveryPath)

	if err := dcli.Before(ctx); err != nil {
		log.Error(err)
		os.Exit(1)
	}

	if err := scli.Before(ctx); err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// setup service upkeep
	up.Init(ctx.Bool("persist"), scli.DefaultSlack)

	if addr != "" {
		log.WithFields(log.Fields{"addr": addr}).Info("API endpoint begin")
		go web.RunAPIEndpoint(addr, stop)
	} else {
		log.Warning("API endpoint disabled")
	}

	if !idle {
		log.Info("Track container life cycle")
		go dkr.RunDockerEvent(stop)
	} else {
		log.Warning("docker event endpoint disabled")
	}

	if addr != "" || !idle {
		<-stop // we should never reach pass this point
	} else {
		log.Warning("nothing to do; quit now")
	}
}
