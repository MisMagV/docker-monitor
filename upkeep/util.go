package upkeep

import (
	log "github.com/Sirupsen/logrus"
	docker "github.com/fsouza/go-dockerclient"

	"time"
)

const (
	DEFAULT_HEARTBEAT = 30 * time.Second
	DEFAULT_TTL       = 35 * time.Second

	DEFAULT_PROBE = 5 * time.Second
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

type Fail struct {
	threshold uint64
	attempts  uint64
}

func NewFail(threshold uint64) *Fail {
	return &Fail{threshold, 0}
}

func (f *Fail) Pass() (ok bool) {
	ok = f.attempts < f.threshold
	f.attempts = 0
	return
}

func (f *Fail) Bad() uint64 {
	f.attempts += 1
	return f.attempts
}

func (f *Fail) Good() uint64 {
	if f.attempts > 0 {
		f.attempts -= 1
	}
	return f.attempts
}
