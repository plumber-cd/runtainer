package golang

import (
	"github.com/plumber-cd/runtainer/discover"
	"github.com/spf13/viper"
)

const (
	defaultGoPath  = "~/go"
	defaultGoCache = "~/.cache/go-build"
)

// Discover specific to Go
func Discover() {
	// get what's already calculated by now
	h, i, v := discover.GetFromViper()

	v.AddEnvVarToDirMountOrExecOrDefault(h, i, "GOPATH", []string{"go", "env", "GOPATH"}, defaultGoPath)
	v.AddEnvVarToDirMountOrExecOrDefault(h, i, "GOCACHE", []string{"go", "env", "GOCACHE"}, defaultGoCache)

	// publish what we've calculated to viper
	viper.Set("volumes", v)
}
