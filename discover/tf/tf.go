package tf

import (
	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/env"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/volumes"
	"github.com/spf13/viper"
)

// Discover specific to AWS
func Discover() {
	log.Debug.Print("Discover Terraform")

	// get what's already calculated by now
	h, e, _, i, v := discover.GetFromViper()

	e.AddEnv(h, &env.DiscoverVariable{Name: "CHECKPOINT_DISABLE"})
	e.AddEnv(h, &env.DiscoverVariable{Name: "TF_LOG"})

	e.AddEnv(h, &env.DiscoverPrefix{Prefix: "TF_VAR_"})

	v.AddHostMount(h, i, "~/.terraform.d/plugin-cache",
		&volumes.DiscoverEnvVar{EnvVar: "TF_PLUGIN_CACHE_DIR"},
		&volumes.DiscoverMirror{},
	)

	log.Debug.Print("Publish to viper")
	viper.Set("env", e)
	viper.Set("volumes", v)
}
