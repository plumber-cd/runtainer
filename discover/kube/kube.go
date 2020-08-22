package kube

import (
	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/log"
	"github.com/spf13/viper"
)

// Discover specific to Kubernetes
func Discover() {
	log.Debug.Print("Discover Kubernetes")

	// get what's already calculated by now
	h, i, v := discover.GetFromViper()

	v.AddEnvVarToFileMountOrDefault(h, i, "KUBECONFIG", "~/.kube")

	log.Debug.Print("Publish to viper")
	viper.Set("volumes", v)
}
