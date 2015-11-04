package main

import (
	docker "github.com/fsouza/go-dockerclient"

	"os"
	"os/signal"
)

func HandleSignal(cli *docker.Client, listener chan *docker.APIEvents) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)
	go func() {
		<-sig // pending on kill/interrupt signal
		cli.RemoveEventListener(listener)
		os.Exit(1)
	}()
}
