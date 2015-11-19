package cmd

import (
	pxy "github.com/jeffjen/docker-ambassador/proxy"
	up "github.com/jeffjen/docker-monitor/upkeep"

	log "github.com/Sirupsen/logrus"
	docker "github.com/fsouza/go-dockerclient"

	"encoding/json"
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

		Proxy = make([]pxy.Info, 0)
	)

	if !up.Validate(info.ID, Srv, Port, Net) {
		return
	}

	if hbStr := info.Config.Labels["heartbeat"]; hbStr == "" {
		Heartbeat = up.DEFAULT_HEARTBEAT
	} else {
		Heartbeat = up.ParseDuration(hbStr, up.DEFAULT_HEARTBEAT)
	}

	if ttlStr := info.Config.Labels["ttl"]; ttlStr == "" {
		TTL = up.DEFAULT_TTL
	} else {
		TTL = up.ParseDuration(ttlStr, up.DEFAULT_TTL)
	}

	if proxySpec := info.Config.Labels["proxy"]; proxySpec != "" {
		if err := json.Unmarshal([]byte(proxySpec), &Proxy); err != nil {
			log.WithFields(log.Fields{"ID": info.ID[:12]}).Warning("reject invalid proxy spec")
			return
		}
	}

	up.MakeService(&up.Service{
		Hb:    Heartbeat,
		TTL:   TTL,
		Id:    info.ID,
		Srv:   Srv,
		Port:  Port,
		Net:   Net,
		Proxy: Proxy,
	})
}
