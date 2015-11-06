package main

import (
	api "github.com/jeffjen/docker-monitor/api"
	disc "github.com/jeffjen/docker-monitor/discovery"
	up "github.com/jeffjen/docker-monitor/upkeep"

	log "github.com/Sirupsen/logrus"
	cli "github.com/codegangsta/cli"
	docker "github.com/fsouza/go-dockerclient"

	"os"
	"time"
)

var (
	Advertise string
	DCli      *docker.Client
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
	}
	app.Action = Monitor
	app.Run(os.Args)
}

func runAPIEndpoint(c *cli.Context) {
	var (
		server = api.GetServer()
		addr   = c.String("addr")
	)
	if addr == "" {
		log.Warning("API endpoint disabled")
		return
	}
	go func() {
		server.Addr = addr
		log.WithFields(log.Fields{"addr": addr}).Info("API endpoint begin")
		log.Fatal(server.ListenAndServe())
	}()
}

func check(c *cli.Context) {
	var (
		hbStr  = c.String("heartbeat")
		ttlStr = c.String("ttl")

		heartbeat time.Duration
		ttl       time.Duration
	)

	runAPIEndpoint(c)

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

	if d, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal(err)
	} else {
		DCli = d
	}

	// docker daemon event source
	src := make(chan *docker.APIEvents, 8)

	// setup signal handler to gracefully quit
	HandleSignal(DCli, src)

	if err := DCli.AddEventListener(src); err != nil {
		log.Fatal(err)
	} else {
		defer DCli.RemoveEventListener(src)
	}

	for event := range src {
		switch event.Status {
		case "start":
			go NewRecord(event.ID)
			break
		case "die":
			go up.ServiceStop(event.ID)
			break
		case "destroy":
			go up.ServiceDie(event.ID)
			break
		}
	}
}

func NewRecord(iden string) {
	if s := up.Get(iden); s != nil {
		if s.Running() {
			s.Stop()
			log.WithFields(log.Fields{"ID": s.Id[:12], "srv": s.Srv}).Warning("inconsistent record")
		}
		s.Start()
		return
	}

	info, _ := DCli.InspectContainer(iden)

	var (
		Srv  = info.Config.Labels["service"]
		Net  = info.NetworkSettings.PortMappingAPI()
		Port = info.Config.Labels["port"]

		Heartbeat time.Duration
		TTL       time.Duration
	)

	if !up.Validate(iden, Srv, Port, Net) {
		return
	}

	if hbStr := info.Config.Labels["heartbeat"]; hbStr == "" {
		Heartbeat = 2 * time.Minute
	} else {
		Heartbeat = up.ParseHearbeat(hbStr)
	}

	if ttlStr := info.Config.Labels["ttl"]; ttlStr == "" {
		TTL = 2*time.Minute + 30*time.Second
	} else {
		TTL = up.ParseTTL(ttlStr)
	}

	up.NewService(Heartbeat, TTL, iden, Srv, info)
}
