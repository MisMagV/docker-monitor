package upkeep

import (
	log "github.com/Sirupsen/logrus"
	docker "github.com/fsouza/go-dockerclient"

	"time"
)

const (
	DEFAULT_HEARTBEAT = 1 * time.Minute
	DEFAULT_TTL       = 1*time.Minute + 5*time.Second

	DEFAULT_PROBE = 15 * time.Second
)

func ParseDuration(s string, df time.Duration) time.Duration {
	if d, err := time.ParseDuration(s); err != nil {
		return df
	} else {
		return d
	}
}

func Validate(iden, srv, port string, network []docker.APIPort) bool {
	if srv == "" {
		log.WithFields(log.Fields{"ID": iden}).Warning("not tracking: service")
		return false
	}
	if port == "" {
		if len(network) == 0 {
			log.WithFields(log.Fields{"ID": iden}).Warning("not tracking: port")
			return false
		}
		open := 0
		for _, net := range network {
			if net.PublicPort != 0 {
				open += 1
			}
		}
		if open == 0 {
			log.WithFields(log.Fields{"ID": iden}).Warning("not tracking: port")
			return false
		}
	}
	return true
}
