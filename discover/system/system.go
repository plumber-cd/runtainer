package system

import (
	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/log"
	"github.com/spf13/viper"
)

// Discover specific to OS
func Discover() {
	log.Debug.Print("Discover System/OS")

	// get what's already calculated by now
	h, i, v := discover.GetFromViper()

	v.AddMirrorHostMount(h, i, "~/.local")
	v.AddMirrorHostMount(h, i, "~/.cache")
	v.AddMirrorHostMount(h, i, "~/.ssh")

	log.Debug.Print("Publish to viper")
	viper.Set("volumes", v)
}
