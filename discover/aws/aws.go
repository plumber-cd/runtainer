package aws

import (
	"github.com/plumber-cd/runtainer/discover"
	"github.com/spf13/viper"
)

// Discover specific to AWS
func Discover() {
	// get what's already calculated by now
	h, i, v := discover.GetFromViper()

	h.AddMirrorEnvVar("AWS_PROFILE")
	h.AddMirrorEnvVar("AWS_DEFAULT_REGION")

	v.AddEnvVarToFileMountOrDefault(h, i, "AWS_SHARED_CREDENTIALS_FILE", "~/.aws")

	// publish what we've calculated to viper
	viper.Set("host", h)
	viper.Set("volumes", v)
}
