package kube

import (
	"golang.org/x/exp/slices"

	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/volumes"
	"github.com/spf13/viper"
)

// Discover specific to Kubernetes
func Discover() {
	disabled := viper.GetStringSlice("discovery.disabled")
	if slices.Contains(disabled, "all") || slices.Contains(disabled, "kube") {
		return
	}

	log.Debug.Print("Discover Kubernetes")

	// get what's already calculated by now
	h, _, _, i, v := discover.GetFromViper()

	v.AddHostMount(h, i, "~/.kube",
		&volumes.DiscoverEnvVar{Config: volumes.DiscoveryConfig{UseParent: true}, EnvVar: "KUBECONFIG"},
		&volumes.DiscoverMirror{},
	)

	log.Debug.Print("Publish to viper")
	viper.Set("volumes", v)
}
