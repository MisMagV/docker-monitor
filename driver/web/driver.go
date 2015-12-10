package web

import (
	d "github.com/jeffjen/docker-monitor/driver"

	log "github.com/Sirupsen/logrus"

	"errors"
	ctx "golang.org/x/net/context"
	http "golang.org/x/net/context/ctxhttp"
	"io"
	"io/ioutil"
	"net/url"
)

var (
	ErrNotReachable = errors.New("web not reachable")
)

type HttpProbe struct {
	url string
}

func (h *HttpProbe) Probe(c ctx.Context) error {
	resp, err := http.Get(c, nil, h.url)
	if err != nil {
		return err
	} else {
		go func() {
			defer resp.Body.Close()
			io.Copy(ioutil.Discard, resp.Body)
		}()
		if resp.StatusCode != 200 {
			return ErrNotReachable
		} else {
			return nil
		}
	}
}

func New(endpoint string) (d.Driver, error) {
	u, _ := url.Parse(endpoint)
	u.Scheme = "http"
	return &HttpProbe{u.String()}, nil
}
