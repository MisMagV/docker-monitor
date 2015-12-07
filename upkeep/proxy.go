package upkeep

import (
	pxy "github.com/jeffjen/docker-ambassador/ambctl/arg"

	log "github.com/Sirupsen/logrus"

	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
)

const (
	DefaultAmbassador = "http://localhost:29091/proxy"
)

func openProxyReq(pflag pxy.Info) {
	var buf = new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(pflag); err != nil {
		log.WithFields(log.Fields{"err": err}).Warning("openProxyReq")
		return
	}
	resp, err := http.Post(DefaultAmbassador, "application/json", buf)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Warning("openProxyReq")
		return
	}
	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)
}

func openProxyConfig(proxycfg string, targets []pxy.Info) {
	// Request to establish proxy port to ambassador
	for _, pxyspec := range targets {
		go openProxyReq(pxyspec)
	}

	var cli = new(http.Client)
	req, err := http.NewRequest("PUT", DefaultAmbassador+"/app-config?key="+proxycfg, nil)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Warning("openProxyConfig")
		return
	}
	resp, err := cli.Do(req)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Warning("openProxyConfig")
		return
	}

	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)
}

// TODO: what should we do to close a proxy req?
