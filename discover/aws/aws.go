package aws

import (
	"github.com/plumber-cd/runtainer/discover"
	"github.com/spf13/viper"
)

// Discover specific to AWS
func Discover() {
	// get what's already calculated by now
	h, i, v := discover.GetFromViper()

	v.AddEnvVarToFileMountOrDefault(h, i, "AWS_SHARED_CREDENTIALS_FILE", "~/.aws")

	// publish what we've calculated to viper
	viper.Set("volumes", v)
}
