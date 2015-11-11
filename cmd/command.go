package cmd

import (
	d "github.com/jeffjen/docker-monitor/driver"
	mgo "github.com/jeffjen/docker-monitor/driver/mongodb"
	r "github.com/jeffjen/docker-monitor/driver/redis"
	web "github.com/jeffjen/docker-monitor/driver/web"
	up "github.com/jeffjen/docker-monitor/upkeep"

	cli "github.com/codegangsta/cli"

	"errors"
)

var (
	Commands = []cli.Command{
		{
			Name:  "web",
			Usage: "Track web service instances by http/https endpoint",
			Flags: append(Flags, cli.StringFlag{
				Name:  "path",
				Value: "/",
				Usage: "Endpoint to probe; default is getting root",
			}),
			Before: endpoint,
			Action: Monitor,
		},
		{
			Name:   "redis",
			Usage:  "Track Redis/Sentinel/Cluster instance",
			Flags:  Flags,
			Before: redis,
			Action: Monitor,
		},
		{
			Name:   "mgo",
			Usage:  "Track mongod/mongos instance",
			Flags:  Flags,
			Before: mongodb,
			Action: Monitor,
		},
		{
			Name:   "tcp",
			Usage:  "Track generic tcp ping endpoint",
			Flags:  Flags,
			Before: tcp,
			Action: Monitor,
		},
	}
)

func endpoint(ctx *cli.Context) error {
	// create closure method for generating new driver for endpoint
	up.AllocDriver = func(endpoint string) (d.Driver, error) {
		return web.New(endpoint)
	}
	return nil
}

func redis(ctx *cli.Context) error {
	// create closure method for generating new driver for endpoint
	up.AllocDriver = func(endpoint string) (d.Driver, error) {
		return r.New(endpoint)
	}
	return nil
}

func mongodb(ctx *cli.Context) error {
	// create closure method for generating new driver for endpoint
	up.AllocDriver = func(endpoint string) (d.Driver, error) {
		return mgo.New(endpoint)
	}
	return nil
}

func tcp(ctx *cli.Context) error {
	// create closure method for generating new driver for endpoint
	up.AllocDriver = func(string) (d.Driver, error) {
		return nil, errors.New("Not yet implemented")
	}
	return nil
}
