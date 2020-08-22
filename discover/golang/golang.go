package golang

import (
	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/log"
	"github.com/spf13/viper"
)

const (
	defaultGoPath  = "~/go"
	defaultGoCache = "~/.cache/go-build"
)

// Discover specific to Go
func Discover() {
	log.Debug.Print("Discover Go")

	// get what's already calculated by now
	h, i, v := discover.GetFromViper()

	v.AddEnvVarToDirMountOrExecOrDefault(h, i, "GOPATH", []string{"go", "env", "GOPATH"}, defaultGoPath)
	v.AddEnvVarToDirMountOrExecOrDefault(h, i, "GOCACHE", []string{"go", "env", "GOCACHE"}, defaultGoCache)

	log.Debug.Print("Publish to viper")
	viper.Set("volumes", v)
}
