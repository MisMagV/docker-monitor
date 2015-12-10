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
	return m.Ping()
}

func New(addr string) (d.Driver, error) {
	sess, err := mgo.Dial(addr)
	if err != nil {
		return nil, err
	} else {
		return &MongoDriver{sess}, nil
	}
}
