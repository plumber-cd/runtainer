package java

import (
	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/volumes"
	"github.com/spf13/viper"
)

// Discover specific to Java
func Discover() {
	log.Debug.Print("Discover Java")

	// get what's already calculated by now
	h, i, v := discover.GetFromViper()

	v.AddHostMount(h, i, "~/.m2",
		&volumes.DiscoverEnvVar{EnvVar: "MAVEN_HOME"},
		&volumes.DiscoverMirror{},
	)

	log.Debug.Print("Publish to viper")
	viper.Set("volumes", v)
}
