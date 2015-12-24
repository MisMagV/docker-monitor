package upkeep

import (
	dri "github.com/jeffjen/docker-monitor/upkeep/driver"
	mgo "github.com/jeffjen/docker-monitor/upkeep/driver/mongodb"
	redis "github.com/jeffjen/docker-monitor/upkeep/driver/redis"
	web "github.com/jeffjen/docker-monitor/upkeep/driver/web"

	log "github.com/Sirupsen/logrus"

	"errors"
)

type Alloc func(string) (dri.Driver, error)

var (
	available = map[string]Alloc{
		// Default noop driver
		"": func(endpoint string) (dri.Driver, error) {
			return &dri.Noop{}, nil
		},

		"redis":    redis.New,
		"sentinel": redis.NewSentinel,
		"mgo":      mgo.New,
		"web":      web.New,
	}

	ErrNoSuchDriver = errors.New("no such driver")
)

func AllocHelper(ptype string) Alloc {
	if d, ok := available[ptype]; ok {
		log.WithFields(log.Fields{"type": ptype}).Debug("AllocHelper")
		return d
	} else {
		log.WithFields(log.Fields{"type": ptype, "err": "not found"}).Warning("AllocHelper")
		return available[""]
	}
}
