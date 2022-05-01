package java

import (
	"golang.org/x/exp/slices"

	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/volumes"
	"github.com/spf13/viper"
)

// Discover specific to Java
func Discover() {
	disabled := viper.GetStringSlice("discovery.disabled")
	if slices.Contains(disabled, "all") || slices.Contains(disabled, "java") {
		return
	}

	log.Debug.Print("Discover Java")

	// get what's already calculated by now
	h, _, _, i, v := discover.GetFromViper()

	v.AddHostMount(h, i, "~/.m2",
		&volumes.DiscoverEnvVar{EnvVar: "MAVEN_HOME"},
		&volumes.DiscoverMirror{},
	)

	log.Debug.Print("Publish to viper")
	viper.Set("volumes", v)
}
