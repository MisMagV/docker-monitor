package upkeep

import (
	push "github.com/jeffjen/go-message/push"

	disc "github.com/jeffjen/go-discovery"
	"github.com/jeffjen/go-libkv/libkv"

	log "github.com/Sirupsen/logrus"
	docker "github.com/fsouza/go-dockerclient"
	ctx "golang.org/x/net/context"

	"encoding/gob"
	"net"
	"reflect"
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

var (
	report push.Pusher
)

func sync(jobId int64) {
	if err := rec.Save(DEFAULT_SYNC_PATH); err != nil {
		log.WithFields(log.Fields{"err": err}).Warning("persist failed")
	} else {
		log.Infof("persist: %v", DEFAULT_SYNC_PATH)
	}
}

func Init(persist bool, pub push.Pusher) {
	cli := GetDockerClient()

	// setup service state change publisher
	if v := reflect.ValueOf(pub); !v.IsValid() || v.IsNil() {
		log.Warning("not publishing service status")
		report = &push.NullPusher{}
	} else {
		report = pub
	}

	// Advertise host URI
	Advertise, _, _ = net.SplitHostPort(disc.Advertise)

	if persist {
		if r, err := libkv.Load(DEFAULT_SYNC_PATH); err != nil {
			log.WithFields(log.Fields{"err": err}).Warning("load failed")
			rec = libkv.NewStore()
		} else {
			rec = r
		}
	} else {
		rec = libkv.NewStore()
	}

	containers, err := cli.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Warning(err)
		return
	}
	for _, c := range containers {
		NewContainerRecord(c.ID)
	}
}

func init() {
	gob.Register(&Service{})
	gob.Register([]docker.APIPort{})

	RootContext, ResetAll = ctx.WithCancel(ctx.Background())
}
