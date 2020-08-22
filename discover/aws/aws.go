package aws

import (
	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/volumes"
	"github.com/spf13/viper"
)

// Discover specific to AWS
func Discover() {
	log.Debug.Print("Discover AWS")

	// get what's already calculated by now
	h, i, v := discover.GetFromViper()

	h.AddMirrorEnvVar("AWS_PROFILE")
	h.AddMirrorEnvVar("AWS_DEFAULT_REGION")

	v.AddHostMount(h, i, "~/.aws",
		&volumes.DiscoverEnvVar{Config: volumes.DiscoveryConfig{UseParent: true}, EnvVar: "AWS_SHARED_CREDENTIALS_FILE"},
		&volumes.DiscoverMirror{},
	)

	log.Debug.Print("Publish to viper")
	viper.Set("host", h)
	viper.Set("volumes", v)
}
