package redis

import (
	d "github.com/jeffjen/docker-monitor/driver"

	ctx "golang.org/x/net/context"
	"gopkg.in/redis.v3"
)

type RedisDriver struct {
	*redis.Client
}

func (r *RedisDriver) Probe(c ctx.Context) error {
	return r.Ping().Err()
}

func New(addr string) (d.Driver, error) {
	cli := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisDriver{cli}, nil
}
