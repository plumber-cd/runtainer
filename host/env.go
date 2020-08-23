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
	log.Debug.Print("Discover Environment")

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
		log.Debug.Print("Checking DOCKER_HOST for dind")
		if dockerHost, dockerHostExists := os.LookupEnv("DOCKER_HOST"); dockerHostExists {
			log.Debug.Printf("DOCKER_HOST env var detected: %s", dockerHost)
			u, err := url.Parse(dockerHost)
			if err != nil {
				log.Stderr.Panic(err)
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

	log.Debug.Print("Discover RT_VAR_* and RT_EVAR_*")
	// now we mirror any env vars that starts with RT_VAR_* to account for any possible user-defined vars
	for _, e := range os.Environ() {
		k := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(k[0], "RT_VAR_") {
			h.AddMirrorEnvVar(k[0])
		}
		if strings.HasPrefix(k[0], "RT_EVAR_") {
			h.AddEnvVarVal(strings.TrimPrefix(k[0], "RT_EVAR_"), k[1])
		}
	}

	log.Debug.Print("Publish to viper")
	viper.Set("host", h)
}

// AddEnvVarVal add an env with a given value to the container
func (h *Host) AddEnvVarVal(e, v string) {
	log.Debug.Printf("Add %s=%s", e, v)
	h.Env = append(h.Env, EnvVar{Name: e, Value: v})
}

// AddEnvVar takes the env var on the host (if defined)
// and explicitly defines it as var:val pair for the container
func (h *Host) AddEnvVar(e string) {
	log.Debug.Printf("Duplicate %s", e)
	if v, ex := os.LookupEnv(e); ex {
		log.Debug.Printf("Duplicate (existing) %s=%s", e, v)
		(*h).Env = append((*h).Env, EnvVar{Name: v, Value: v})
	}
}

// AddMirrorEnvVar just like AddEnvVar but does not explicitly defines a value,
// just mirrors it from the host
func (h *Host) AddMirrorEnvVar(e string) {
	log.Debug.Printf("Mirror %s", e)
	if _, ex := os.LookupEnv(e); ex {
		log.Debug.Printf("Mirror (existing) %s", e)
		(*h).Env = append((*h).Env, EnvVar{Name: e})
	}
}
