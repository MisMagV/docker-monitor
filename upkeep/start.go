package upkeep

import (
	docker "github.com/fsouza/go-dockerclient"
	dri "github.com/jeffjen/docker-monitor/driver"
	disc "github.com/jeffjen/go-discovery"
	"github.com/jeffjen/go-libkv/libkv"
	timer "github.com/jeffjen/go-libkv/timer"

	log "github.com/Sirupsen/logrus"

	"encoding/gob"
	"net"
	"time"
)

const (
	DEFAULT_SYNC_PATH  = "/tmp"
	DEFAULT_SYNC_CYCLE = 2 * time.Minute
)

var (
	Sched *timer.Timer

	rec *libkv.Store

	AllocDriver func(string) (dri.Driver, error) = nil

	Advertise string
)

func noop(string) (dri.Driver, error) {
	return &dri.Noop{}, nil
}

func sync(jobId int64) {
	if err := rec.Save(DEFAULT_SYNC_PATH); err != nil {
		log.WithFields(log.Fields{"err": err}).Warning("persist failed")
	} else {
		log.Infof("persist: %v", DEFAULT_SYNC_PATH)
	}
}

func Init(persist bool) {
	var err error

	Advertise, _, _ = net.SplitHostPort(disc.Advertise)

	if AllocDriver == nil {
		AllocDriver = noop // default safety net for driver maker
	}

	Sched = timer.NewTimer()

	Sched.Tic() // start the scheduler, don't ever stop

	if persist {
		if rec, err = libkv.Load(DEFAULT_SYNC_PATH); err != nil {
			log.WithFields(log.Fields{"err": err}).Warning("load failed")
		}
		for _, k := range rec.Key() {
			MakeService(rec.Get(k).(*Service))
		}
		Sched.RepeatFunc(DEFAULT_SYNC_CYCLE, 1, sync)
	} else {
		rec = libkv.NewStore()
	}
}

func init() {
	gob.Register(&Service{})
	gob.Register([]docker.APIPort{})
}
