package helm

import (
	"golang.org/x/exp/slices"

	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/host"
	"github.com/plumber-cd/runtainer/image"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/volumes"
	"github.com/spf13/viper"
	"helm.sh/helm/v3/pkg/helmpath"
)

// Discover specific to Helm
func Discover() {
	disabled := viper.GetStringSlice("discovery.disabled")
	if slices.Contains(disabled, "all") || slices.Contains(disabled, "helm") {
		return
	}

	log.Debug.Print("Discover Helm")

	// get what's already calculated by now
	h, _, _, i, v := discover.GetFromViper()

	v.AddHostMount(h, i, "~/.helm",
		&volumes.DiscoverMirror{},
	)

	v.AddHostMount(h, i, "~/.cache/helm",
		&volumes.DiscoverCallback{Callback: func(_ host.Host, _ image.Image, _ string) (bool, string) {
			return true, helmpath.CachePath("")
		}},
	)
	v.AddHostMount(h, i, "~/.config/helm",
		&volumes.DiscoverCallback{Callback: func(_ host.Host, _ image.Image, _ string) (bool, string) {
			return true, helmpath.ConfigPath("")
		}},
	)

	log.Debug.Print("Publish to viper")
	viper.Set("volumes", v)
}
