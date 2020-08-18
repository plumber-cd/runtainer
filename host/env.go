package host

import (
	"net/url"
	"os"
	"strings"

	"github.com/plumber-cd/runtainer/log"
	"github.com/spf13/viper"
)

func discoverEnv() {
	h := viper.Get("host").(Host)

	h.Env = append(h.Env, EnvVar{Name: "RT_HOST_NAME", Value: h.Name})
	h.Env = append(h.Env, EnvVar{Name: "RT_HOST_USER", Value: h.User})
	h.Env = append(h.Env, EnvVar{Name: "RT_HOST_HOME", Value: h.Home})
	h.Env = append(h.Env, EnvVar{Name: "RT_HOST_CWD", Value: h.Cwd})

	h.mirrorVar("AWS_PROFILE")
	h.mirrorVar("AWS_DEFAULT_REGION")

	if s := os.Getenv("DOCKER_HOST"); s != "" {
		log.Debug.Printf("DOCKER_HOST env var detected: %s", s)
		u, err := url.Parse(s)
		if err != nil {
			log.Error.Panic(err)
		}
		if u.Hostname() == "localhost" || strings.HasPrefix(u.Hostname(), "127.") {
			internal := "host.docker.internal"
			log.Debug.Printf("DOCKER_HOST env var was pointing to localhost (%s), patch it to %s", u.Hostname(), internal)
			u.Host = internal + ":" + u.Port()
		} else {
			log.Debug.Printf("Assuming %s is external address", u.Hostname())
		}
		log.Debug.Printf("Adding DOCKER_HOST=%s", u.String())
		h.Env = append(h.Env, EnvVar{Name: "DOCKER_HOST", Value: u.String()})
	}

	for _, e := range os.Environ() {
		k := strings.SplitN(e, "=", 2)[0]
		if strings.HasPrefix(k, "RT_") {
			h.mirrorVar(k)
		}
	}
	viper.Set("host", h)
}

func (h *Host) mirrorVar(v string) {
	if s := os.Getenv(v); s != "" {
		(*h).Env = append((*h).Env, EnvVar{Name: v, Value: s})
	}
}
