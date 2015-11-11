package upkeep

import (
	disc "github.com/jeffjen/docker-monitor/discovery"
	d "github.com/jeffjen/docker-monitor/driver"
	"github.com/jeffjen/go-libkv/libkv"
	timer "github.com/jeffjen/go-libkv/timer"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"
	docker "github.com/fsouza/go-dockerclient"
	ctx "golang.org/x/net/context"

	"encoding/gob"
	"fmt"
	"path"
	"strings"
	"time"
)

const (
	DEFAULT_HEARTBEAT = 2 * time.Minute
	DEFAULT_TTL       = 2*time.Minute + 5*time.Second

	DEFAULT_SYNC_PATH  = "/tmp"
	DEFAULT_SYNC_CYCLE = 2 * time.Minute
)

var (
	Sched *timer.Timer

	rec *libkv.Store

	AllocDriver func(string) (d.Driver, error) = nil
)

func sync(jobId int64) {
	if err := rec.Save(DEFAULT_SYNC_PATH); err != nil {
		log.Warningf("unable to persist: %v", err)
	} else {
		log.Infof("persist: %v", DEFAULT_SYNC_PATH)
	}
}

func noop(string) (d.Driver, error) {
	return &d.Noop{}, nil
}

func Init(persist bool) {
	var err error

	if AllocDriver == nil {
		AllocDriver = noop // default safety net for driver maker
	}

	Sched = timer.NewTimer()

	Sched.Tic() // start the scheduler, don't ever stop

	if persist {
		if rec, err = libkv.Load(DEFAULT_SYNC_PATH); err != nil {
			log.Warningf("unable to load: %v", err)
		}
		for _, k := range rec.Key() {
			MakeService(rec.Get(k).(*Service))
		}
		Sched.RepeatFunc(DEFAULT_SYNC_CYCLE, 1, sync)
	} else {
		rec = libkv.NewStore()
	}
}

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

func MakeService(s *Service) {
	s.f = log.Fields{"ID": s.Id[:12], "srv": s.Srv, "heartbeat": s.Hb, "ttl": s.TTL}

	if len(s.Net) > 0 {
		key := make([]string, 0)
		for _, p := range s.Net {
			if p.PublicPort != 0 && p.IP == "0.0.0.0" {
				key = append(key, fmt.Sprintf("%s:%d", disc.Advertise, p.PublicPort))
			}
		}
		if len(key) == 1 {
			s.key = path.Join(s.Srv, strings.Join(key, ","))
		} else {
			log.WithFields(log.Fields{"ID": s.Id[:12], "Net": s.Net}).Warning("refuse; 0 or too many port")
			return
		}
	} else if s.Port != "" {
		s.key = path.Join(s.Srv, fmt.Sprintf("%s:%s", disc.Advertise, s.Port))
	}

	s.opts = &etcd.SetOptions{TTL: s.TTL}

	if s.Start() {
		rec.Set(s.Id, s)
	} else {
		log.WithFields(log.Fields{"ID": s.Id[:12]}).Error("not tracking: probe failed")
	}
}

func NewService(heartbeat, ttl time.Duration, iden, service, port string, net []docker.APIPort) (s *Service) {
	s = &Service{
		Hb:   heartbeat,
		TTL:  ttl,
		Id:   iden,
		Srv:  service,
		Port: port,
		Net:  net,
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

	Id   string           `json: "ContainerID"`
	Srv  string           `json: "Service"`
	Port string           `json: "Port"`
	Net  []docker.APIPort `json: "Net"`

	kAPI etcd.KeysAPI     `json:-`
	key  string           `json:-`
	opts *etcd.SetOptions `json:-`

	jobId int64 `json:-`

	f log.Fields `json:-`

	driver d.Driver `json:-`
	fail   int      `json:-`
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

func (s *Service) Upkeep() {
	if _, err := s.kAPI.Set(ctx.Background(), s.key, s.Id, s.opts); err != nil {
		log.WithFields(s.f).Warning(err)
	} else {
		log.WithFields(s.f).Info("up")
		s.opts.PrevExist = etcd.PrevExist
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

func init() {
	gob.Register(&Service{})
	gob.Register([]docker.APIPort{})
}
