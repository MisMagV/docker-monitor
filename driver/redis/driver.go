package redis

import (
	d "github.com/jeffjen/docker-monitor/driver"
	disc "github.com/jeffjen/go-discovery"
	node "github.com/jeffjen/go-discovery/info"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"
	ctx "golang.org/x/net/context"
	"gopkg.in/redis.v3"

	"errors"
	"net"
	"path"
	"strings"
	"time"
)

type event struct {
	msg *redis.Message
	err error
}

type masterEvent struct {
	old string
	nue string
}

func doReceiveMessage(pubsub *redis.PubSub) (v <-chan *event) {
	resp := make(chan *event, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.WithFields(log.Fields{"err": r}).Debug("-sentinel")
			}
		}()
		msg, err := pubsub.ReceiveMessage()
		resp <- &event{msg, err}
	}()
	return resp
}

type RedisDriver struct {
	*redis.Client
	abortSentinel ctx.CancelFunc
	kAPI          etcd.KeysAPI
}

func (r *RedisDriver) Probe(c ctx.Context) error {
	resp := make(chan error, 1)
	go func() {
		resp <- r.Ping().Err()
	}()
	select {
	case err := <-resp:
		return err
	case <-c.Done():
		return c.Err()
	}
}

func (r *RedisDriver) Close() error {
	if r.abortSentinel != nil {
		r.abortSentinel()
	}
	return nil
}

func (r *RedisDriver) publishMaster(c ctx.Context) (MasterInfo chan<- *masterEvent) {
	input := make(chan *masterEvent, 1)
	go func() {
		m, ok := <-input
		if !ok {
			log.WithFields(log.Fields{"err": "gone"}).Debug("-sentinel")
			return
		}

		wk, abort := ctx.WithTimeout(c, 250*time.Millisecond)
		defer abort()

		_, err := r.kAPI.Delete(wk, m.old, nil)
		if err != nil {
			log.WithFields(log.Fields{"err": err, "o": m.old, "n": m.nue}).Debug("-switch")
		}
		_, err = r.kAPI.Set(wk, m.nue, node.MetaData, nil)
		if err != nil {
			log.WithFields(log.Fields{"err": err, "o": m.old, "n": m.nue}).Debug("-switch")
		}
		if err == nil {
			log.WithFields(log.Fields{"o": m.old, "n": m.nue}).Warning("+switch")
		}
	}()
	return input
}

func (r *RedisDriver) follow(c ctx.Context) {
	go func() {
		var (
			MasterInfo chan<- *masterEvent = nil

			pubsub *redis.PubSub
		)

		for err := errors.New("not ready"); err != nil; {
			pubsub, err = r.Subscribe("+elected-leader", "+switch-master")
			if err != nil {
				log.WithFields(log.Fields{"err": err}).Warning("-sentinel")
				time.Sleep(1 * time.Second)
			}
		}
		defer func() {
			pubsub.Close()
			if MasterInfo != nil {
				close(MasterInfo)
			}
		}()

		log.Info("+sentinel")
		for yay := true; yay; {
			resp := doReceiveMessage(pubsub)
			select {
			case evt := <-resp:
				if evt.err != nil {
					log.WithFields(log.Fields{"err": evt.err}).Warning("-sentinel")
					break
				}
				switch evt.msg.Channel {
				case "+elected-leader":
					if MasterInfo != nil {
						close(MasterInfo)
					}
					MasterInfo = r.publishMaster(c)
					break
				case "+switch-master":
					if MasterInfo == nil {
						break // don't process: not from eleceted leader
					}
					parts := strings.Split(evt.msg.Payload, " ")
					MasterInfo <- &masterEvent{
						old: path.Join(parts[0], net.JoinHostPort(parts[1], parts[2])),
						nue: path.Join(parts[0], net.JoinHostPort(parts[3], parts[4])),
					}
					// Reset plumming
					MasterInfo = nil
					break
				}
			case <-c.Done():
				yay = false
			}
		}
	}()
}

func New(addr string) (d.Driver, error) {
	cli := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisDriver{cli, nil, nil}, nil
}

func NewSentinel(addr string) (d.Driver, error) {
	cli := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	wk, abort := ctx.WithCancel(ctx.Background())
	driver := &RedisDriver{
		cli,
		abort,
		etcd.NewKeysAPI(disc.NewDiscovery()),
	}
	driver.follow(wk)
	return driver, nil
}
