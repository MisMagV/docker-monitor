package upkeep

import (
	pxy "github.com/jeffjen/docker-ambassador/proxy"

	log "github.com/Sirupsen/logrus"

	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
)

func openProxyReq(pflag pxy.Info) {
	var buf = new(bytes.Buffer)
	json.NewEncoder(buf).Encode(pflag)
	resp, err := http.Post("http://localhost:29091/proxy", "application/json", buf)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Warning("openProxyReq")
	}
	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)
}

// TODO: what should we do to close a proxy req?
