package web

import (
	d "github.com/jeffjen/docker-monitor/driver"

	"errors"
	"fmt"
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
		return ErrNotReachable
	} else {
		go func() {
			defer resp.Body.Close()
			io.Copy(ioutil.Discard, resp.Body)
		}()
		if resp.StatusCode != 200 {
			return fmt.Errorf("%d: %s", resp.StatusCode, h.url)
		} else {
			return nil
		}
	}
}

func (h *HttpProbe) Close() error {
	return nil
}

func New(endpoint string) (d.Driver, error) {
	u, _ := url.Parse(endpoint)
	u.Scheme = "http"
	return &HttpProbe{u.String()}, nil
}
