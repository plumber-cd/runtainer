package system

import (
	"os"
	"path/filepath"

	"golang.org/x/exp/slices"

	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/env"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/volumes"
	"github.com/spf13/viper"
)

// Discover specific to OS
func Discover() {
	disabled := viper.GetStringSlice("discovery.disabled")
	if slices.Contains(disabled, "all") || slices.Contains(disabled, "system") {
		return
	}

	log.Debug.Print("Discover System/OS")

	// get what's already calculated by now
	h, e, _, i, v := discover.GetFromViper()

	if !slices.Contains(disabled, "system.local") {
		v.AddHostMount(h, i, "~/.local",
			&volumes.DiscoverMirror{},
		)
	}
	if !slices.Contains(disabled, "system.cache") {
		v.AddHostMount(h, i, "~/.cache",
			&volumes.DiscoverMirror{},
		)
	}
	if !slices.Contains(disabled, "system.ssh") {
		v.AddHostMount(h, i, "~/.ssh",
			&volumes.DiscoverMirror{},
		)
	}
	if !slices.Contains(disabled, "system.gnupg") {
		v.AddHostMount(h, i, "~/.gnupg",
			&volumes.DiscoverMirror{},
		)
	}

	if !slices.Contains(disabled, "system.ssh-auth-sock") {
		sshAuthSock, okSshAuthSock := os.LookupEnv("SSH_AUTH_SOCK")
		if okSshAuthSock {
			e.AddEnv(h, &env.DiscoverValue{
				Name:  "SSH_AUTH_SOCK",
				Value: "/rt-host-ssh-auth-sock/" + filepath.Base(sshAuthSock),
			})
			v.AddHostMount(h, i, "/rt-host-ssh-auth-sock",
				&volumes.DiscoverEnvVar{
					Config: volumes.DiscoveryConfig{
						UseParent: true,
					},
					EnvVar: "SSH_AUTH_SOCK",
				},
			)
		}
	}

	log.Debug.Print("Publish to viper")
	viper.Set("env", e)
	viper.Set("volumes", v)
}
