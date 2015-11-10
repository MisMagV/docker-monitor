package cmd

import (
	disc "github.com/jeffjen/docker-monitor/discovery"
	up "github.com/jeffjen/docker-monitor/upkeep"

	log "github.com/Sirupsen/logrus"
	cli "github.com/codegangsta/cli"

	"os"
	"time"
)

func check(c *cli.Context) error {
	var (
		hbStr   = c.String("heartbeat")
		ttlStr  = c.String("ttl")
		persist = c.Bool("persist")

		heartbeat time.Duration
		ttl       time.Duration
	)

	if disc.Advertise = c.String("advertise"); disc.Advertise == "" {
		cli.ShowAppHelp(c)
		log.Error("Required flag --advertise missing")
		os.Exit(1)
	}

	heartbeat, err := time.ParseDuration(hbStr)
	if err != nil {
		log.Fatal(err)
	}
	ttl, err = time.ParseDuration(ttlStr)
	if err != nil {
		log.Fatal(err)
	}

	if pos := c.Args(); len(pos) != 1 {
		cli.ShowAppHelp(c)
		log.Error("Required arguemnt DISCOVERY_URI")
		os.Exit(1)
	} else {
		disc.Discovery = pos[0]
	}

	// register monitor instance
	disc.Register(heartbeat, ttl)

	// setup service upkeep
	up.Init(persist)

	log.WithFields(log.Fields{
		"advertise": disc.Advertise,
		"discovery": disc.Discovery,
		"heartbeat": heartbeat,
		"ttl":       ttl,
	}).Info("begin monitor")

	return nil
}

func Monitor(ctx *cli.Context) {
	var (
		addr = ctx.String("addr")
		idle = ctx.Bool("idle")

		stop = make(chan struct{}, 1)
	)

	check(ctx) // launch system init

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
