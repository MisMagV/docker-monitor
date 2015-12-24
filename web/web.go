package web

import (
	api "github.com/jeffjen/docker-monitor/web/api"
	dsc "github.com/jeffjen/go-discovery/info"

	log "github.com/Sirupsen/logrus"
)

func init() {
	vmux := &api.VarServeMux{}
	vmux.HandleFunc("/s/([a-z0-9]+)/register", api.Register)
	vmux.HandleFunc("/s/([a-z0-9]+)/update", api.Update)

	api.GetServeMux().HandleFunc("/info", dsc.Info)
	api.GetServeMux().Handle("/s/", vmux)
}

func RunAPIEndpoint(addr string, stop chan<- struct{}) {
	defer close(stop)

	server := api.GetServer()

	server.Addr = addr
	log.Error(server.ListenAndServe())
}
