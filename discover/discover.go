package discover

import (
	"github.com/plumber-cd/runtainer/env"
	"github.com/plumber-cd/runtainer/host"
	"github.com/plumber-cd/runtainer/image"
	"github.com/plumber-cd/runtainer/volumes"
	"github.com/spf13/viper"
)

// GetFromViper loads all available and calculated data from viper
func GetFromViper() (host.Host, env.Env, image.Image, volumes.Volumes) {
	h := viper.Get("host").(host.Host)
	e := viper.Get("env").(env.Env)
	i := viper.Get("image").(image.Image)
	v := viper.Get("volumes").(volumes.Volumes)
	return h, e, i, v
}
