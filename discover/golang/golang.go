package golang

import (
	"github.com/plumber-cd/runtainer/discover"
	"github.com/spf13/viper"
)

// Discover specific to Go
func Discover() {
	// get what's already calculated by now
	h, i, v := discover.GetFromViper()

	v.AddEnvVarToDirMountOrDefault(h, i, "GOPATH", "~/go")

	// publish what we've calculated to viper
	viper.Set("volumes", v)
}
