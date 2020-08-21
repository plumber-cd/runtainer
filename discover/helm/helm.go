package helm

import (
	"runtime"

	"github.com/plumber-cd/runtainer/discover"
	"github.com/spf13/viper"
)

// Discover specific to Helm
func Discover() {
	// get what's already calculated by now
	h, i, v := discover.GetFromViper()

	v.AddMirrorHostMount(h, i, "~/.helm")

	switch runtime.GOOS {
	case "darwin":
		v.AddHostMount(h, i, "~/Library/Preferences/helm", "~/.config/helm")
		v.AddHostMount(h, i, "~/Library/Caches/helm", "~/.cache/helm")
	default:
		v.AddMirrorHostMount(h, i, "~/.config/helm")
		v.AddMirrorHostMount(h, i, "~/.cache/helm")
	}

	// publish what we've calculated to viper
	viper.Set("volumes", v)
}
