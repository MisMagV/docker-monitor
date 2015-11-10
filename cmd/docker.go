package cmd

import (
	up "github.com/jeffjen/docker-monitor/upkeep"

	log "github.com/Sirupsen/logrus"
	docker "github.com/fsouza/go-dockerclient"

	"time"
)

var (
	dCli *docker.Client
)

func runDockerEvent(stop chan<- struct{}) {
	defer close(stop)

	if d, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal(err)
	} else {
		dCli = d
	}

	// docker daemon event source
	src := make(chan *docker.APIEvents, 8)

	if err := dCli.AddEventListener(src); err != nil {
		log.Fatal(err)
	} else {
		defer dCli.RemoveEventListener(src)
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
			log.WithFields(log.Fields{"ID": s.Id[:12]}).Warning("inconsistent record")
		}
		s.Start()
		return
	}

	info, _ := dCli.InspectContainer(iden)

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
		Heartbeat = up.DEFAULT_HEARTBEAT
	} else {
		Heartbeat = up.ParseHearbeat(hbStr)
	}

	if ttlStr := info.Config.Labels["ttl"]; ttlStr == "" {
		TTL = up.DEFAULT_TTL
	} else {
		TTL = up.ParseTTL(ttlStr)
	}

	up.NewService(Heartbeat, TTL, iden, Srv, Port, Net)
}
