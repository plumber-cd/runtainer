package host

import (
	"net/url"
	"os"
	"strings"

	"github.com/plumber-cd/runtainer/log"
	"github.com/spf13/viper"
)

// discoverEnv use to define RunTainer specific known environment variables
func discoverEnv() {
	h := viper.Get("host").(Host)

	// just define soma standard host facts as env variables
	h.AddEnvVarVal("RT_HOST_NAME", h.Name)
	h.AddEnvVarVal("RT_HOST_USER", h.User)
	h.AddEnvVarVal("RT_HOST_HOME", h.Home)
	h.AddEnvVarVal("RT_HOST_CWD", h.Cwd)

	// Unless explicitly disabled, we need to account for possible docker in docker calls.
	// We will look into DOCKER_HOST and if it pointing to the local network interface,
	// we need to translate it to host.docker.internal so it's accessible from within the container.
	if !viper.GetBool("dind") {
		if dockerHost, dockerHostExists := os.LookupEnv("DOCKER_HOST"); dockerHostExists {
			log.Debug.Printf("DOCKER_HOST env var detected: %s", dockerHost)
			u, err := url.Parse(dockerHost)
			if err != nil {
				log.Error.Panic(err)
			}
			if u.Hostname() == "localhost" || strings.HasPrefix(u.Hostname(), "127.") {
				internal := "host.docker.internal"
				log.Debug.Printf("DOCKER_HOST env var was pointing to localhost (%s), patch it to %s", u.Hostname(), internal)
				u.Host = internal + ":" + u.Port()
			}
			log.Debug.Printf("Adding DOCKER_HOST=%s", u.String())
			h.AddEnvVarVal("DOCKER_HOST", u.String())
		}
	}

	// now we mirror any env vars that starts with RT_VAR_* to account for any possible user-defined vars
	for _, e := range os.Environ() {
		k := strings.SplitN(e, "=", 2)[0]
		if strings.HasPrefix(k, "RT_VAR_") {
			h.AddMirrorEnvVar(k)
		}
	}

	// also pass any env vars that starts with RT_EVAR_* removing the prefix
	// to allow non-prefixed env variables in the container
	for _, e := range os.Environ() {
		k := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(k[0], "RT_EVAR_") {
			h.AddEnvVarVal(strings.TrimPrefix(k[0], "RT_EVAR_"), k[1])
		}
	}

	// publish what we've calculated so far to viper
	viper.Set("host", h)
}

// AddEnvVarVal add an env with a given value to the container
func (h *Host) AddEnvVarVal(e, v string) {
	h.Env = append(h.Env, EnvVar{Name: e, Value: v})
}

// AddEnvVar takes the env var on the host (if defined)
// and explicitly defines it as var:val pair for the container
func (h *Host) AddEnvVar(e string) {
	if v, ex := os.LookupEnv(e); ex {
		(*h).Env = append((*h).Env, EnvVar{Name: v, Value: v})
	}
}

// AddMirrorEnvVar just like AddEnvVar but does not explicitly defines a value,
// just mirrors it from the host
func (h *Host) AddMirrorEnvVar(e string) {
	if _, ex := os.LookupEnv(e); ex {
		(*h).Env = append((*h).Env, EnvVar{Name: e})
	}
}
