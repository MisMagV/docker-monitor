package api

import (
	up "github.com/jeffjen/docker-monitor/upkeep"

	log "github.com/Sirupsen/logrus"

	"fmt"
	"net/http"
)

func common(m string, r *http.Request, args []string) (string, error) {
	if r.Method != m {
		return "", fmt.Errorf("method not allowed")
	}
	if err := r.ParseForm(); err != nil {
		return "", fmt.Errorf("unable to process argument")
	}
	if args[0] == "" {
		return "", fmt.Errorf("bad request")
	}
	return args[0], nil
}

func Register(w http.ResponseWriter, r *http.Request, args []string) {
	iden, err := common("POST", r, args)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Warning("register")
		http.Error(w, "Method not allowed", 403)
		return
	}
	w.Write([]byte("processing"))

	go up.NewContainerRecord(iden)
}

func Update(w http.ResponseWriter, r *http.Request, args []string) {
	iden, err := common("PUT", r, args)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Warning("update")
		http.Error(w, "Method not allowed", 403)
		return
	}

	service := up.Get(iden)
	if service == nil {
		http.NotFound(w, r)
		return
	}

	w.Write([]byte("updating"))

	go func() {
		if srv := r.Form.Get("srv"); srv != "" {
			service.Srv = srv
		}
		if port := r.Form.Get("port"); port != "" {
			service.Port = port
		}
		up.Place(service)
	}()
}
