package cmd

import (
	"github.com/plumber-cd/runtainer/discover/aws"
	"github.com/plumber-cd/runtainer/discover/golang"
	"github.com/plumber-cd/runtainer/discover/java"
	"github.com/plumber-cd/runtainer/discover/kube"
	"github.com/plumber-cd/runtainer/discover/system"
	"github.com/plumber-cd/runtainer/host"
	"github.com/plumber-cd/runtainer/image"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/volumes"
)

func discover(imageName string) {
	log.Debug.Print("Start discovery routine")

	host.DiscoverHost()
	image.DiscoverImage(imageName)
	volumes.DiscoverVolumes()

	system.Discover()
	aws.Discover()
	kube.Discover()
	golang.Discover()
	java.Discover()
}
