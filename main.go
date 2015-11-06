package main

import (
	disc "github.com/jeffjen/docker-monitor/discovery"

	log "github.com/Sirupsen/logrus"
	cli "github.com/codegangsta/cli"

	"os"
	"time"
)

func main() {
	app := cli.NewApp()
	app.Name = "docker-monitor"
	app.Usage = "Monitor docker events and report to discovery service"
	app.Authors = []cli.Author{
		cli.Author{"Yi-Hung Jen", "yihungjen@gmail.com"},
	}
	app.Flags = []cli.Flag{
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
	}
	app.Action = Monitor
	app.Run(os.Args)
}

func check(c *cli.Context) {
	var (
		hbStr  = c.String("heartbeat")
		ttlStr = c.String("ttl")

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

	log.WithFields(log.Fields{
		"advertise": disc.Advertise,
		"discovery": disc.Discovery,
		"heartbeat": heartbeat,
		"ttl":       ttl,
	}).Info("begin monitor")

	return
}

func Monitor(ctx *cli.Context) {
	check(ctx)

	var (
		addr = ctx.String("addr")
		idle = ctx.Bool("idle")

		stop = make(chan struct{}, 1)
	)

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
