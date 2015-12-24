package docker

import (
	up "github.com/jeffjen/docker-monitor/upkeep"

	log "github.com/Sirupsen/logrus"
	dkr "github.com/fsouza/go-dockerclient"
)

func RunDockerEvent(stop chan<- struct{}) {
	defer close(stop)

	// docker daemon event source
	src := make(chan *dkr.APIEvents, 8)

	// docker client
	dCli := up.GetDockerClient()

	if err := dCli.AddEventListener(src); err != nil {
		log.Fatal(err)
	} else {
		defer dCli.RemoveEventListener(src)
	}

	for event := range src {
		switch event.Status {
		case "start":
			go up.NewContainerRecord(event.ID)
			break
		case "die":
			go up.Suspend(event.ID)
			break
		case "destroy":
			go up.Unregister(event.ID)
			break
		}
	}
}
