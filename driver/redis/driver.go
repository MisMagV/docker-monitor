package redis

import (
	d "github.com/jeffjen/docker-monitor/driver"

	"gopkg.in/redis.v3"
)

type RedisDriver struct {
	*redis.Client
}

func (r *RedisDriver) Probe() error {
	return r.Ping().Err()
}

func (r *RedisDriver) Close() {
}

func New(addr string) (d.Driver, error) {
	cli := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisDriver{cli}, nil
}
