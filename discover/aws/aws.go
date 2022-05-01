package aws

import (
	"golang.org/x/exp/slices"

	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/env"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/volumes"
	"github.com/spf13/viper"
)

// Discover specific to AWS
func Discover() {
	disabled := viper.GetStringSlice("discovery.disabled")
	if slices.Contains(disabled, "all") || slices.Contains(disabled, "aws") {
		return
	}

	log.Debug.Print("Discover AWS")

	// get what's already calculated by now
	h, e, _, i, v := discover.GetFromViper()

	e.AddEnv(h, &env.DiscoverVariable{Name: "AWS_ACCESS_KEY_ID"})
	e.AddEnv(h, &env.DiscoverVariable{Name: "AWS_SECRET_ACCESS_KEY"})
	e.AddEnv(h, &env.DiscoverVariable{Name: "AWS_SESSION_TOKEN"})
	e.AddEnv(h, &env.DiscoverVariable{Name: "AWS_PROFILE"})
	e.AddEnv(h, &env.DiscoverVariable{Name: "AWS_ROLE_SESSION_NAME"})
	e.AddEnv(h, &env.DiscoverVariable{Name: "AWS_DEFAULT_REGION"})
	e.AddEnv(h, &env.DiscoverVariable{Name: "AWS_STS_REGIONAL_ENDPOINTS"})
	e.AddEnv(h, &env.DiscoverVariable{Name: "AWS_SDK_LOAD_CONFIG"})

	v.AddHostMount(h, i, "~/.aws",
		&volumes.DiscoverEnvVar{Config: volumes.DiscoveryConfig{UseParent: true}, EnvVar: "AWS_SHARED_CREDENTIALS_FILE"},
		&volumes.DiscoverMirror{},
	)

	log.Debug.Print("Publish to viper")
	viper.Set("env", e)
	viper.Set("volumes", v)
}
