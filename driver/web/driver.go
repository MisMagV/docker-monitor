package web

import (
	d "github.com/jeffjen/docker-monitor/driver"
)

type HttpProbe struct{}

func (h *HttpProbe) Probe() error {
	return nil
}

func (h *HttpProbe) Close() {
}

func New(addr string) (d.Driver, error) {
	return &HttpProbe{}, nil
}
