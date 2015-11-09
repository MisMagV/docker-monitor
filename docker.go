package main

import (
	docker "github.com/fsouza/go-dockerclient"
	up "github.com/jeffjen/docker-monitor/upkeep"

	log "github.com/Sirupsen/logrus"

	"time"
)

var (
	DCli *docker.Client
)

func runDockerEvent(stop chan<- struct{}) {
	defer close(stop)

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
			go newRecord(event.ID)
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

func newRecord(iden string) {
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

	up.NewService(Heartbeat, TTL, iden, Srv, Port, info)
}
