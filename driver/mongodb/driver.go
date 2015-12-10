package mongodb

import (
	d "github.com/jeffjen/docker-monitor/driver"

	ctx "golang.org/x/net/context"
	"gopkg.in/mgo.v2"
)

type MongoDriver struct {
	*mgo.Session
}

func (m *MongoDriver) Probe(c ctx.Context) error {
	resp := make(chan error, 1)
	go func() {
		resp <- m.Ping()
	}()
	select {
	case err := <-resp:
		return err
	case <-c.Done():
		return c.Err()
	}
}

func New(addr string) (d.Driver, error) {
	sess, err := mgo.Dial(addr)
	if err != nil {
		return nil, err
	} else {
		return &MongoDriver{sess}, nil
	}
}
