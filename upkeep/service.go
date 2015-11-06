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

func ParseHearbeat(s string) time.Duration {
	if hb, err := time.ParseDuration(s); err != nil {
		return 2 * time.Minute
	} else {
		return hb
	}
}

func ParseTTL(s string) time.Duration {
	if ttl, err := time.ParseDuration(s); err != nil {
		return 2*time.Minute + 30*time.Second
	} else {
		return ttl
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

func MakeService(s *Service) {
	if port := s.C.Config.Labels["port"]; port == "" {
		key := make([]string, 0)
		net := s.C.NetworkSettings.PortMappingAPI()
		for _, p := range net {
			if p.PublicPort != 0 {
				key = append(key, fmt.Sprintf("%s:%d", disc.Advertise, p.PublicPort))
			}
		}
		s.key = path.Join(s.Srv, strings.Join(key, ","))
	} else {
		s.key = path.Join(s.Srv, fmt.Sprintf("%s:%s", disc.Advertise, port))
	}

	s.opts = &etcd.SetOptions{TTL: s.TTL}

	if rec.Set(s.Id, s) {
		s.Start()
	}
}

func NewService(heartbeat, ttl time.Duration, iden, service string, container *docker.Container) (s *Service) {
	s = &Service{
		Hb:  heartbeat,
		TTL: ttl,
		Id:  iden,
		Srv: service,
		C:   container,
	}
	MakeService(s)
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
	Hb  time.Duration `json: "Heartbeat"`
	TTL time.Duration `json: "TTL"`

	Id  string            `json: "ContainerID"`
	Srv string            `json: "Service"`
	C   *docker.Container `json:-`

	kAPI etcd.KeysAPI     `json:-`
	key  string           `json:-`
	opts *etcd.SetOptions `json:-`

	jobId int64 `json:-`
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

func (s *Service) Update() {
	Sched.Cancel(s.jobId)
	log.WithFields(log.Fields{"ID": s.Id[:12], "srv": s.Srv, "ttl": s.TTL}).Info("update")
	MakeService(s)
}

func (s *Service) Start() {
	s.kAPI = etcd.NewKeysAPI(disc.NewDiscovery())
	s.Upkeep()
	s.opts.PrevExist = etcd.PrevExist
	s.jobId = Sched.Repeat(s.Hb, 1, s)
}

func (s *Service) Stop() {
	Sched.Cancel(s.jobId)
	s.kAPI = nil
	s.jobId = -1
	s.opts.PrevExist = etcd.PrevIgnore
	log.WithFields(log.Fields{"ID": s.Id[:12], "srv": s.Srv, "ttl": s.TTL}).Info("down")
}

func (s *Service) Running() bool {
	return s.jobId != -1
}

func init() {
	Sched.Tic() // start the scheduler, don't ever stop
}
