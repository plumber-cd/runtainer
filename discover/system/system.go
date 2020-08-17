package system

import (
	"github.com/plumber-cd/runtainer/discover"
	"github.com/spf13/viper"
)

// Discover specific to OS
func Discover() {
	// get what's already calculated by now
	h, i, v := discover.GetFromViper()

	v.AddMirrorHostMount(h, i, "~/.local")
	v.AddMirrorHostMount(h, i, "~/.cache")
	v.AddMirrorHostMount(h, i, "~/.ssh")

	// publish what we've calculated to viper
	viper.Set("volumes", v)
}
