package system

import (
	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/volumes"
	"github.com/spf13/viper"
)

// Discover specific to OS
func Discover() {
	log.Debug.Print("Discover System/OS")

	// get what's already calculated by now
	h, _, _, i, v := discover.GetFromViper()

	v.AddHostMount(h, i, "~/.local",
		&volumes.DiscoverMirror{},
	)
	v.AddHostMount(h, i, "~/.cache",
		&volumes.DiscoverMirror{},
	)
	v.AddHostMount(h, i, "~/.ssh",
		&volumes.DiscoverMirror{},
	)

	log.Debug.Print("Publish to viper")
	viper.Set("volumes", v)
}
