package api

import (
	pxy "github.com/jeffjen/ambd/ambctl/arg"
	up "github.com/jeffjen/docker-monitor/upkeep"
	d "github.com/jeffjen/go-discovery/info"

	log "github.com/Sirupsen/logrus"
	docker "github.com/fsouza/go-dockerclient"

	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func common(m string, r *http.Request, args []string) (*docker.Container, error) {
	if r.Method != m {
		return nil, fmt.Errorf("method not allowed")
	}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("unable to process argument")
	}

	var iden string = args[0]

	if dCli, err := docker.NewClientFromEnv(); err != nil {
		return nil, err
	} else {
		return dCli.InspectContainer(iden)
	}
}

func register(w http.ResponseWriter, r *http.Request, args []string) {
	info, err := common("POST", r, args)
	if err != nil {
		http.Error(w, "Method not allowed", 403)
		return
	}

	if s := up.Get(info.ID); s != nil {
		w.Write([]byte("exist"))
		return
	}

	var (
		Srv  = r.Form.Get("srv")
		Net  = info.NetworkSettings.PortMappingAPI()
		Port = r.Form.Get("port")

		Heartbeat time.Duration
		TTL       time.Duration

		ProbeHeartbeat time.Duration
		ProbeType      = info.Config.Labels["probetype"]
		ProbeEndpoint  = info.Config.Labels["probe_endpoint"]

		Proxy    = make([]pxy.Info, 0)
		ProxyCfg string
	)

	if !up.Validate(info.ID, Srv, Port, Net) {
		http.Error(w, "Bad Request", 400)
		return
	}

	if hbStr := r.Form.Get("hb"); hbStr == "" {
		Heartbeat = up.DEFAULT_HEARTBEAT
	} else {
		Heartbeat = up.ParseDuration(hbStr, up.DEFAULT_HEARTBEAT)
	}

	if phbStr := info.Config.Labels["probefreq"]; phbStr == "" {
		ProbeHeartbeat = up.DEFAULT_PROBE
	} else {
		ProbeHeartbeat = up.ParseDuration(phbStr, up.DEFAULT_PROBE)
	}

	if ttlStr := r.Form.Get("ttl"); ttlStr == "" {
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
	ProxyCfg = info.Config.Labels["proxycfg"]

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

	w.Write([]byte("ok"))
}

func update(w http.ResponseWriter, r *http.Request, args []string) {
	info, err := common("PUT", r, args)
	if err != nil {
		http.Error(w, "Method not allowed", 403)
		return
	}

	service := up.Get(info.ID)
	if service == nil {
		http.NotFound(w, r)
		return
	}

	if srv := r.Form.Get("srv"); srv != "" {
		service.Srv = srv
	}

	if port := r.Form.Get("port"); port != "" {
		service.Port = port
	}

	up.Place(service)

	w.Write([]byte("ok"))
}

func init() {
	mux = http.NewServeMux()
	s = &http.Server{Handler: mux}

	vmux := &VarServeMux{}
	vmux.HandleFunc("/s/([a-z0-9]+)/register", register)
	vmux.HandleFunc("/s/([a-z0-9]+)/update", update)

	mux := GetServeMux()
	mux.HandleFunc("/info", d.Info)
	mux.Handle("/s/", vmux)
}
