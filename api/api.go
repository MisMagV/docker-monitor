package api

import (
	up "github.com/jeffjen/docker-monitor/upkeep"

	_ "github.com/Sirupsen/logrus"
	docker "github.com/fsouza/go-dockerclient"

	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

var (
	VERSION = os.Getenv("VERSION")

	BUILD = os.Getenv("BUILD")

	NODE_NAME = os.Getenv("NODE_NAME")

	NODE_REGION = os.Getenv("NODE_REGION")

	NODE_AVAIL_ZONE = os.Getenv("NODE_AVAIL_ZONE")
)

type serverInfo struct {
	Version   string `json:"version"`
	Build     string `json:"build"`
	Node      string `json:"node"`
	Region    string `json:"region"`
	Zone      string `json:"avail_zone"`
	Timestamp string `json:"current_time"`
}

func getInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(serverInfo{
		Version:   VERSION,
		Build:     BUILD,
		Node:      NODE_NAME,
		Region:    NODE_REGION,
		Zone:      NODE_AVAIL_ZONE,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

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
	)

	if !up.Validate(info.ID, Srv, Port, Net) {
		http.Error(w, "Bad Request", 400)
		return
	}

	if hbStr := r.Form.Get("hb"); hbStr == "" {
		Heartbeat = up.DEFAULT_HEARTBEAT
	} else {
		Heartbeat = up.ParseHearbeat(hbStr)
	}

	if ttlStr := r.Form.Get("ttl"); ttlStr == "" {
		TTL = up.DEFAULT_TTL
	} else {
		TTL = up.ParseTTL(ttlStr)
	}

	up.NewService(Heartbeat, TTL, info.ID, Srv, Port, Net)

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

	if hbStr := r.Form.Get("hb"); hbStr != "" {
		service.Hb = up.ParseHearbeat(hbStr)
	}

	if ttlStr := r.Form.Get("ttl"); ttlStr != "" {
		service.TTL = up.ParseTTL(ttlStr)
	}

	service.Update()

	w.Write([]byte("ok"))
}

func init() {
	mux = http.NewServeMux()
	s = &http.Server{Handler: mux}

	vmux := &VarServeMux{}
	vmux.HandleFunc("/s/([a-z0-9]+)/register", register)
	vmux.HandleFunc("/s/([a-z0-9]+)/update", update)

	mux := GetServeMux()
	mux.HandleFunc("/info", getInfo)
	mux.Handle("/s/", vmux)
}
