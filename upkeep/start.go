package upkeep

import (
	disc "github.com/jeffjen/go-discovery"
	"github.com/jeffjen/go-libkv/libkv"

	log "github.com/Sirupsen/logrus"
	docker "github.com/fsouza/go-dockerclient"
	ctx "golang.org/x/net/context"

	"encoding/gob"
	"net"
	"time"
)

const (
	DEFAULT_SYNC_PATH  = "/tmp"
	DEFAULT_SYNC_CYCLE = 2 * time.Minute
)

var (
	rec *libkv.Store

	Advertise string
)

type RunningRecord struct {
	Srv   string
	Abort ctx.CancelFunc
}

var (
	RootContext ctx.Context
	ResetAll    ctx.CancelFunc

	Record = make(map[string]*RunningRecord)
)

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

	if persist {
		if rec, err = libkv.Load(DEFAULT_SYNC_PATH); err != nil {
			log.WithFields(log.Fields{"err": err}).Warning("load failed")
		}
		for k := range rec.IterateR() {
			service, ok := k.X.(*Service)
			if ok {
				Register(service)
			}
		}
	} else {
		rec = libkv.NewStore()
	}
}

func init() {
	gob.Register(&Service{})
	gob.Register([]docker.APIPort{})

	RootContext, ResetAll = ctx.WithCancel(ctx.Background())
}
