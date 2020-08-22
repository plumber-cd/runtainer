package java

import (
	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/log"
	"github.com/spf13/viper"
)

// Discover specific to Java
func Discover() {
	log.Debug.Print("Discover Java")

	// get what's already calculated by now
	h, i, v := discover.GetFromViper()

	v.AddEnvVarToDirMountOrDefault(h, i, "MAVEN_HOME", "~/.m2")

	log.Debug.Print("Publish to viper")
	viper.Set("volumes", v)
}
