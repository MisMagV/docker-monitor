package upkeep

import (
	disc "github.com/jeffjen/docker-monitor/discovery"
	"github.com/jeffjen/go-libkv/libkv"
	timer "github.com/jeffjen/go-libkv/timer"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"
	docker "github.com/fsouza/go-dockerclient"
	ctx "golang.org/x/net/context"

	"fmt"
	"path"
	"strings"
	"time"
)

var (
	Sched = timer.NewTimer()

	rec = libkv.NewStore()
)

func NewService(heartbeat, ttl time.Duration, iden, service string, container *docker.Container) (s *Service) {
	s = &Service{
		Hb:  heartbeat,
		TTL: ttl,
		Id:  iden,
		Srv: service,
		C:   container,
	}

	if port := container.Config.Labels["port"]; port == "" {
		key := make([]string, 0)
		net := container.NetworkSettings.PortMappingAPI()
		for _, p := range net {
			if p.PublicPort != 0 {
				key = append(key, fmt.Sprintf("%s:%s:%d", p.Type, disc.Advertise, p.PublicPort))
			}
		}
		s.key = path.Join(service, strings.Join(key, ","))
	} else {
		scheme := container.Config.Labels["scheme"]
		if scheme == "" {
			scheme = "tcp"
		}
		s.key = path.Join(service, fmt.Sprintf("%s:%s:%s", scheme, disc.Advertise, port))
	}

	s.opts = &etcd.SetOptions{TTL: ttl}

	if rec.Set(iden, s) {
		s.Start()
	}
	return
}

func ServiceStop(iden string) (s *Service) {
	if s, _ := rec.Get(iden).(*Service); s != nil && s.Running() {
		s.Stop()
	}
	return
}

func ServiceDie(iden string) (s *Service) {
	if s, _ := rec.Get(iden).(*Service); s != nil {
		if s.Running() {
			s.Stop()
		}
		rec.Del(iden)
		log.WithFields(log.Fields{"ID": s.Id[:12], "srv": s.Srv}).Info("die")
	}
	return
}

func Get(iden string) (s *Service) {
	s, _ = rec.Get(iden).(*Service)
	return
}

type Service struct {
	Hb  time.Duration
	TTL time.Duration

	Id  string
	Srv string
	C   *docker.Container

	jobId int64 // required info for cancel

	kAPI etcd.KeysAPI // for recored upkeep
	key  string
	opts *etcd.SetOptions
}

func (s *Service) Done(jobId int64) {
	s.Upkeep()
}

func (s *Service) Upkeep() {
	f := log.Fields{"ID": s.Id[:12], "srv": s.Srv, "heartbeat": s.Hb, "ttl": s.TTL}
	if _, err := s.kAPI.Set(ctx.Background(), s.key, s.Id, s.opts); err != nil {
		log.WithFields(f).Warning(err)
	} else {
		log.WithFields(f).Info("up")
	}
}

func (s *Service) Start() {
	s.kAPI = etcd.NewKeysAPI(disc.NewDiscovery())
	s.Upkeep()
	s.jobId = Sched.Repeat(s.Hb, 1, s)
}

func (s *Service) Stop() {
	Sched.Cancel(s.jobId)
	s.kAPI = nil
	s.jobId = -1
	log.WithFields(log.Fields{"ID": s.Id[:12], "srv": s.Srv, "ttl": s.TTL}).Info("down")
}

func (s *Service) Running() bool {
	return s.jobId != -1
}

func init() {
	Sched.Tic() // start the scheduler, don't ever stop
}
