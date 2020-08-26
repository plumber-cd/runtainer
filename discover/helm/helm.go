package helm

import (
	"image"

	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/host"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/volumes"
	"github.com/spf13/viper"
	"helm.sh/helm/v3/pkg/helmpath"
)

// Discover specific to Helm
func Discover() {
	log.Debug.Print("Discover Helm")

	// get what's already calculated by now
	h, _, i, v := discover.GetFromViper()

	v.AddHostMount(h, i, "~/.helm",
		&volumes.DiscoverMirror{},
	)

	v.AddHostMount(h, i, "~/.cache/helm",
		&volumes.DiscoverCallback{Callback: func(h host.Host, i image.Image, dest string) (bool, string) {
			return true, helmpath.CachePath("")
		}},
	)
	v.AddHostMount(h, i, "~/.config/helm",
		&volumes.DiscoverCallback{Callback: func(h host.Host, i image.Image, dest string) (bool, string) {
			return true, helmpath.ConfigPath("")
		}},
	)

	log.Debug.Print("Publish to viper")
	viper.Set("volumes", v)
}
