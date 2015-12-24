package upkeep

import (
	pxy "github.com/jeffjen/ambd/ambctl/arg"

	log "github.com/Sirupsen/logrus"
	docker "github.com/fsouza/go-dockerclient"
	dri "github.com/jeffjen/docker-monitor/upkeep/driver"
	disc "github.com/jeffjen/go-discovery"
	node "github.com/jeffjen/go-discovery/info"

	etcd "github.com/coreos/etcd/client"
	ctx "golang.org/x/net/context"

	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"
)

var (
	dCli *docker.Client
)

func GetDockerClient() *docker.Client {
	if dCli == nil {
		if d, err := docker.NewClientFromEnv(); err != nil {
			//log.Fatal(err)
		} else {
			dCli = d
		}
	}
	return dCli
}

func init() {
	var level = os.Getenv("LOG_LEVEL")
	switch level {
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
		break
	case "INFO":
		log.SetLevel(log.InfoLevel)
		break
	case "WARNING":
		log.SetLevel(log.WarnLevel)
		break
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
		break
	case "FATAL":
		log.SetLevel(log.FatalLevel)
		break
	case "PANIC":
		log.SetLevel(log.PanicLevel)
		break
	default:
		log.SetLevel(log.InfoLevel)
		break
	}
}

const (
	UpkeepTimeout = 3 * time.Second

	ProbeTimeout = 1 * time.Second

	MaxFailAttemps = 3
)

const (
	ServiceUp          = "up"
	ServiceUnavailable = "unavailable"
	ServiceDown        = "down"
	ServiceRemoved     = "die"
)

type Service struct {
	State string `json:"State"`

	Hb  time.Duration `json: "Heartbeat"`
	TTL time.Duration `json: "TTL"`

	PHb           time.Duration `json: "ProbeHeartbeat"`
	ProbeType     string        `json: "ProbeType"`
	ProbeEndpoint string        `json: "ProbeEndpoint"`

	Id       string           `json: "ContainerID"`
	Srv      string           `json: "Service"`
	Port     string           `json: "Port"`
	Net      []docker.APIPort `json: "Net"`
	Proxy    []pxy.Info       `json: "Proxy"`
	ProxyCfg string           `json: "ProxyCfg"`

	Key []string `json: "Key"`
}

func pushReport(serv *srv) {
	pushlog := log.WithFields(log.Fields{"srv": serv.Srv, "state": serv.State})
	go func() {
		wk, cancel := ctx.WithTimeout(ctx.Background(), 5*time.Second)
		defer cancel()
		if err := report.Push(wk, serv); err != nil {
			pushlog.WithFields(log.Fields{"err": err}).Warning("push")
		}
	}()
}

type srv struct {
	*Service // embedded type

	kAPI etcd.KeysAPI
	opts *etcd.SetOptions

	driver dri.Driver
}

func (serv *srv) keep(c ctx.Context) (err error) {
	for _, k := range serv.Key {
		_, err = serv.kAPI.Set(c, k, node.MetaData, serv.opts)
		if err != nil {
			break
		}
	}
	preState := serv.State
	if err != nil {
		serv.State = ServiceUnavailable
		serv.opts.PrevExist = etcd.PrevIgnore
		pushReport(serv)
	} else {
		serv.State = ServiceUp
		serv.opts.PrevExist = etcd.PrevExist
		if preState != ServiceUp {
			pushReport(serv)
		}
	}
	return
}

func (serv *srv) probe(c ctx.Context) (err error) {
	err = serv.driver.Probe(c)
	log.WithFields(log.Fields{"err": err, "srv": serv.Srv}).Debug("probe")
	return
}

func (serv *srv) urk() {
	serv.State = ServiceUnavailable
	serv.opts.PrevExist = etcd.PrevIgnore
	pushReport(serv)
}

func (serv *srv) down() {
	serv.State = ServiceDown
	serv.opts.PrevExist = etcd.PrevIgnore
	pushReport(serv)
}

func (serv *srv) die() {
	serv.State = ServiceRemoved
	pushReport(serv)
}

func Get(iden string) (s *Service) {
	s, _ = rec.Get(iden).(*Service)
	return
}

func Place(service *Service) {
	if r, ok := Record[service.Id]; ok {
		r.Abort()
	}
	rec.Set(service.Id, service)
	Register(service)
}

func Register(service *Service) {
	alloc := AllocHelper(service.ProbeType)
	service.Key = make([]string, 0)

	serv := &srv{
		service,
		etcd.NewKeysAPI(disc.NewDiscovery()),
		&etcd.SetOptions{TTL: service.TTL, PrevExist: etcd.PrevIgnore},
		nil,
	}

	logger := log.WithFields(log.Fields{
		"ID": serv.Id[:12], "srv": serv.Srv, "heartbeat": serv.Hb, "ttl": serv.TTL,
	})

	// Advertise Key on the discovery service
	if serv.Port != "" {
		serv.Key = append(serv.Key, fmt.Sprintf("%s/%s:%s", serv.Srv, Advertise, serv.Port))
	} else if len(serv.Net) > 0 {
		serv.Key = make([]string, 0)
		for _, p := range serv.Net {
			if p.PublicPort != 0 && p.IP == "0.0.0.0" {
				serv.Key = append(serv.Key, fmt.Sprintf("%s/%s:%d", serv.Srv, Advertise, p.PublicPort))
			}
		}
	}

	var endpoint string
	if serv.ProbeEndpoint == "" {
		endpoint = path.Base(serv.Key[0])
	} else {
		endpoint = path.Join(path.Base(serv.Key[0]), serv.ProbeEndpoint)
	}

	// TODO:  setup driver for probing
	driver, drr := alloc(endpoint)
	if drr != nil {
		logger.WithFields(log.Fields{"err": drr}).Error("-register")
		return
	}
	serv.driver = driver
	logger.Debug("+register")

	wk, abort := ctx.WithCancel(RootContext)
	go func() {
		defer serv.driver.Close()

		// Request to establish proxy port to ambassador
		openProxyConfig(serv.ProxyCfg, serv.Proxy)

		// setup work cycle
		heartbeat, probe := time.NewTicker(serv.Hb), time.NewTicker(serv.PHb)
		defer func() {
			heartbeat.Stop()
			probe.Stop()
		}()

		logger.Info("start")
		func() {
			d, abort := ctx.WithTimeout(wk, UpkeepTimeout)
			if err := serv.keep(d); err != nil {
				logger.WithFields(log.Fields{"err": err, "state": serv.opts.PrevExist}).Error("-up")
			} else {
				logger.Info("+up")
			}
			abort() // release
		}()

		var chk = NewFail(MaxFailAttemps)
		for yay := true; yay; {
			select {
			case <-heartbeat.C:
				if !chk.Pass() {
					serv.urk()
					logger.Error("!up")
				} else {
					d, abort := ctx.WithTimeout(wk, UpkeepTimeout)
					if err := serv.keep(d); err != nil {
						logger.WithFields(log.Fields{"err": err, "state": serv.opts.PrevExist}).Error("-up")
					} else {
						logger.Info("+up")
					}
					abort() // release
				}

			case <-probe.C:
				d, abort := ctx.WithTimeout(wk, ProbeTimeout)
				if err := serv.probe(d); err != nil {
					count := chk.Bad()
					logger.WithFields(log.Fields{"err": err, "fail": count}).Warning("-probe")
				} else {
					chk.Good()
					logger.Debug("+probe")
				}
				abort() // release

			case <-wk.Done():
				serv.down()
				logger.Warning("down")
				yay = false
			}
		}
	}()

	Record[serv.Id] = &RunningRecord{serv.Srv, abort} // register abort function for this service
}

func NewContainerRecord(iden string) {
	logger := log.WithFields(log.Fields{"ID": iden[:12]})

	if s := Get(iden); s != nil {
		Register(s)
		return
	}

	info, err := dCli.InspectContainer(iden)
	if err != nil {
		logger.WithFields(log.Fields{"err": err}).Warning("NewRecord")
		return
	}
	if !info.State.Running {
		logger.WithFields(log.Fields{"err": "not running"}).Warning("NewRecord")
		return
	}

	var (
		Srv  = info.Config.Labels["service"]
		Net  = info.NetworkSettings.PortMappingAPI()
		Port = info.Config.Labels["port"]

		Heartbeat time.Duration
		TTL       time.Duration

		ProbeHeartbeat time.Duration
		ProbeType      = info.Config.Labels["probe_type"]
		ProbeEndpoint  = info.Config.Labels["probe_endpoint"]

		Proxy    = make([]pxy.Info, 0)
		ProxyCfg = info.Config.Labels["proxycfg"]
	)

	if !Validate(info.ID, Srv, Port, Net) {
		return
	}

	if hbStr := info.Config.Labels["heartbeat"]; hbStr == "" {
		Heartbeat = DEFAULT_HEARTBEAT
	} else {
		Heartbeat = ParseDuration(hbStr, DEFAULT_HEARTBEAT)
	}

	if phbStr := info.Config.Labels["probe_heartbeat"]; phbStr == "" {
		ProbeHeartbeat = DEFAULT_PROBE
	} else {
		ProbeHeartbeat = ParseDuration(phbStr, DEFAULT_PROBE)
	}

	if ttlStr := info.Config.Labels["ttl"]; ttlStr == "" {
		TTL = DEFAULT_TTL
	} else {
		TTL = ParseDuration(ttlStr, DEFAULT_TTL)
	}

	if proxySpec := info.Config.Labels["proxy"]; proxySpec != "" {
		if err := json.Unmarshal([]byte(proxySpec), &Proxy); err != nil {
			logger.WithFields(log.Fields{"err": err}).Warning("NewRecord")
			return
		}
	}

	Place(&Service{
		State:         ServiceUp,
		Hb:            Heartbeat,
		TTL:           TTL,
		PHb:           ProbeHeartbeat,
		ProbeType:     ProbeType,
		ProbeEndpoint: ProbeEndpoint,
		Id:            info.ID,
		Srv:           Srv,
		Port:          Port,
		Net:           Net,
		Proxy:         Proxy,
		ProxyCfg:      ProxyCfg,
	})
}

func Suspend(iden string) {
	if r, ok := Record[iden]; ok {
		r.Abort()
	}
}

func Unregister(iden string) {
	logger := log.WithFields(log.Fields{"ID": iden[:12]})

	if r, ok := Record[iden]; ok {
		delete(Record, iden)
		r.Abort()
		go func() {
			serv := &srv{rec.Get(iden).(*Service), nil, nil, nil}
			serv.die()
			rec.Del(iden)
		}()
		logger.WithFields(log.Fields{"srv": r.Srv}).Warning("die")
	}
}
