package cmd

import (
	pxy "github.com/jeffjen/ambd/ambctl/arg"
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
			go up.Suspend(event.ID)
			break
		case "destroy":
			go up.Unregister(event.ID)
			break
		}
	}
}

func newRecord(iden string) {
	if s := up.Get(iden); s != nil {
		up.Register(s)
		return
	}

	info, _ := dCli.InspectContainer(iden)

	var (
		Srv  = info.Config.Labels["service"]
		Net  = info.NetworkSettings.PortMappingAPI()
		Port = info.Config.Labels["port"]

		Heartbeat time.Duration
		TTL       time.Duration

		ProbeHeartbeat time.Duration
		ProbeType      = info.Config.Labels["probe_type"]
		ProbeEndpoint  = info.Config.Labels["probe_endpoint"]

		Proxy    = make([]pxy.Info, 0)
		ProxyCfg = info.Config.Labels["proxycfg"]
	)

	if !up.Validate(info.ID, Srv, Port, Net) {
		return
	}

	if hbStr := info.Config.Labels["heartbeat"]; hbStr == "" {
		Heartbeat = up.DEFAULT_HEARTBEAT
	} else {
		Heartbeat = up.ParseDuration(hbStr, up.DEFAULT_HEARTBEAT)
	}

	if phbStr := info.Config.Labels["probe_heartbeat"]; phbStr == "" {
		ProbeHeartbeat = up.DEFAULT_PROBE
	} else {
		ProbeHeartbeat = up.ParseDuration(phbStr, up.DEFAULT_PROBE)
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

	up.Place(&up.Service{
		State:         up.ServiceUp,
		Hb:            Heartbeat,
		TTL:           TTL,
		PHb:           ProbeHeartbeat,
		ProbeType:     ProbeType,
		ProbeEndpoint: ProbeEndpoint,
		Id:            info.ID,
		Srv:           Srv,
		Port:          Port,
		Net:           Net,
		Proxy:         Proxy,
		ProxyCfg:      ProxyCfg,
	})
}
