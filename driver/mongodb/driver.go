package mongodb

import (
	d "github.com/jeffjen/docker-monitor/driver"

	"gopkg.in/mgo.v2"
)

type MongoDriver struct {
	*mgo.Session
}

func (m *MongoDriver) Probe() error {
	return m.Ping()
}

func (m *MongoDriver) Close() {
}

func New(addr string) (d.Driver, error) {
	sess, err := mgo.Dial(addr)
	if err != nil {
		return nil, err
	} else {
		return &MongoDriver{sess}, nil
	}
}
