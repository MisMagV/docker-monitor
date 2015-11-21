package upkeep

import (
	pxy "github.com/jeffjen/docker-ambassador/proxy"
	dri "github.com/jeffjen/docker-monitor/driver"
	disc "github.com/jeffjen/go-discovery"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"
	docker "github.com/fsouza/go-dockerclient"
	ctx "golang.org/x/net/context"

	"fmt"
	"path"
	"strings"
	"time"
)

func MakeService(s *Service) {
	s.f = log.Fields{"ID": s.Id[:12], "srv": s.Srv, "heartbeat": s.Hb, "ttl": s.TTL}

	// Determine how this service is found by other service
	if s.Port != "" {
		s.key = path.Join(s.Srv, fmt.Sprintf("%s:%s", Advertise, s.Port))
	} else if len(s.Net) > 0 {
		key := make([]string, 0)
		for _, p := range s.Net {
			if p.PublicPort != 0 && p.IP == "0.0.0.0" {
				key = append(key, fmt.Sprintf("%s:%d", Advertise, p.PublicPort))
			}
		}
		if len(key) == 1 {
			s.key = path.Join(s.Srv, strings.Join(key, ","))
		} else {
			log.WithFields(log.Fields{"ID": s.Id[:12], "Net": s.Net}).Warning("refuse; 0 or too many port")
			return
		}
	}

	// Request to establish proxy port to ambassador
	for _, pspec := range s.Proxy {
		go openProxyReq(pspec)
	}

	s.opts = &etcd.SetOptions{TTL: s.TTL}

	if s.Start() {
		rec.Set(s.Id, s)
	} else {
		log.WithFields(log.Fields{"ID": s.Id[:12]}).Error("not tracking: probe failed")
	}
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

	Id    string           `json: "ContainerID"`
	Srv   string           `json: "Service"`
	Port  string           `json: "Port"`
	Net   []docker.APIPort `json: "Net"`
	Proxy []pxy.Info       `json: "Proxy"`

	kAPI etcd.KeysAPI     `json:-`
	key  string           `json:-`
	opts *etcd.SetOptions `json:-`

	jobId int64 `json:-`

	f log.Fields `json:-`

	driver dri.Driver `json:-`
	fail   int        `json:-`
}

func (s *Service) Done(jobId int64) {
	if s.Probe() != nil {
		s.fail += 1
		log.WithFields(s.f).Warning("probe missed")
		return
	}
	if s.fail > 3 {
		s.Stop()
		return
	} else {
		s.fail = 0
		s.Upkeep()
	}
}

func (s *Service) Probe() error {
	return s.driver.Probe()
}

func (s *Service) keep() error {
	_, err := s.kAPI.Set(ctx.Background(), s.key, s.Id, s.opts)
	if err == nil {
		s.opts.PrevExist = etcd.PrevExist
	} else {
		s.opts.PrevExist = etcd.PrevIgnore
	}
	return err
}

func (s *Service) Upkeep() {
	if err := s.keep(); err != nil {
		nrr, ok := err.(etcd.Error)
		if !ok {
			log.WithFields(s.f).Error(err)
			return
		}
		if nrr.Code != etcd.ErrorCodeKeyNotFound {
			log.WithFields(s.f).Error(nrr)
			return
		}
		// Last resort: set through
		if err = s.keep(); err != nil {
			log.WithFields(s.f).Error(err)
		}
	} else {
		log.WithFields(s.f).Info("up")
	}
}

func (s *Service) Update() {
	Sched.Cancel(s.jobId)
	log.WithFields(s.f).Info("update")
	MakeService(s)
}

func (s *Service) Start() bool {
	// FIXME: we probably need a allow specific port for probing
	if s.driver, _ = AllocDriver(path.Base(s.key)); s.driver == nil {
		return false
	}

	s.kAPI = etcd.NewKeysAPI(disc.NewDiscovery())

	s.Upkeep()
	if s.Probe() != nil {
		s.fail += 1
		log.WithFields(s.f).Warning("probe missed")
	}

	s.jobId = Sched.Repeat(s.Hb, 1, s)

	return true
}

func (s *Service) Stop() {
	Sched.Cancel(s.jobId)
	if s.driver != nil {
		s.driver.Close()
		s.driver = nil
	}
	s.kAPI = nil
	s.jobId = -1
	s.opts.PrevExist = etcd.PrevIgnore
	log.WithFields(s.f).Info("down")
}

func (s *Service) Running() bool {
	return s.jobId != -1
}
