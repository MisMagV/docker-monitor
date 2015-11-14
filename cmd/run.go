package cmd

import (
	disc "github.com/jeffjen/docker-monitor/discovery"
	up "github.com/jeffjen/docker-monitor/upkeep"

	log "github.com/Sirupsen/logrus"
	cli "github.com/codegangsta/cli"

	"os"
)

func Monitor(ctx *cli.Context) {
	var (
		addr = ctx.String("addr")
		idle = ctx.Bool("idle")

		stop = make(chan struct{}, 1)
	)

	disc.Check(ctx) // launch system init

	// setup service upkeep
	up.Init(ctx.Bool("persist"))

	if addr != "" {
		log.WithFields(log.Fields{"addr": addr}).Info("API endpoint begin")
		go runAPIEndpoint(addr, stop)
	} else {
		log.Warning("API endpoint disabled")
	}

	if !idle {
		log.WithFields(log.Fields{"addr": os.Getenv("DOCKER_HOST")}).Info("Track container life cycle")
		go runDockerEvent(stop)
	} else {
		log.Warning("docker event endpoint disabled")
	}

	if addr != "" || !idle {
		<-stop // we should never reach pass this point
	} else {
		log.Warning("nothing to do; quit now")
	}
}
