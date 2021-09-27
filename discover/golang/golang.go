package golang

import (
	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/volumes"
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
	h, _, _, i, v := discover.GetFromViper()

	v.AddHostMount(h, i, defaultGoPath,
		&volumes.DiscoverEnvVar{EnvVar: "GOPATH"},
		&volumes.DiscoverExec{Args: []string{"go", "env", "GOPATH"}},
		&volumes.DiscoverMirror{},
	)
	v.AddHostMount(h, i, defaultGoCache,
		&volumes.DiscoverEnvVar{EnvVar: "GOCACHE"},
		&volumes.DiscoverExec{Args: []string{"go", "env", "GOCACHE"}},
		&volumes.DiscoverMirror{},
	)

	log.Debug.Print("Publish to viper")
	viper.Set("volumes", v)
}
