package driver

import (
	log "github.com/Sirupsen/logrus"
	ctx "golang.org/x/net/context"

	"io"
	"os"
)

type Driver interface {
	io.Closer
	Probe(c ctx.Context) error
}

type Noop struct{}

func (n *Noop) Probe(c ctx.Context) error {
	return nil
}

func (n *Noop) Close() error {
	return nil
}

func init() {
	var level = os.Getenv("LOG_LEVEL")
	switch level {
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
		break
	case "INFO":
		log.SetLevel(log.InfoLevel)
		break
	case "WARNING":
		log.SetLevel(log.WarnLevel)
		break
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
		break
	case "FATAL":
		log.SetLevel(log.FatalLevel)
		break
	case "PANIC":
		log.SetLevel(log.PanicLevel)
		break
	default:
		log.SetLevel(log.InfoLevel)
		break
	}
}
