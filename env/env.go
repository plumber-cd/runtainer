package env

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/plumber-cd/runtainer/host"
	"github.com/plumber-cd/runtainer/log"
	"github.com/spf13/viper"
)

// DiscoveryConfig various configuration for discovery
type DiscoveryConfig struct {
	CopyValue bool
}

func (dc *DiscoveryConfig) string() string {
	return fmt.Sprintf("DiscoveryConfig{CopyValue: %v}", dc.CopyValue)
}

// Discover is an interface for various discoverers
type Discover interface {
	// Discover must return true only if anything discovered
	discover(h host.Host) (found bool, vars map[string]interface{})
}

// DiscoverValue don't even tries to discover anything, just forcibly adds key:val pair.
// As it may never existed on the host, Config.CopyValue will be ignored and value always will be set.
type DiscoverValue struct {
	Config      DiscoveryConfig
	Name, Value string
}

func (d *DiscoverValue) string() string {
	return fmt.Sprintf("DiscoverValue{Name: %s, Value: %s} with %s", d.Name, d.Value, d.Config.string())
}

func (d *DiscoverValue) discover(_ host.Host) (bool, map[string]interface{}) {
	log.Debug.Print(d.string())
	return true, map[string]interface{}{d.Name: d.Value}
}

// DiscoverVariable basically is a default fallback discoverer.
// It will look for exact variable name.
type DiscoverVariable struct {
	Config DiscoveryConfig
	Name   string
}

func (d *DiscoverVariable) string() string {
	return fmt.Sprintf("DiscoverVariable{Name: %s} with %s", d.Name, d.Config.string())
}

func (d *DiscoverVariable) discover(_ host.Host) (bool, map[string]interface{}) {
	log.Debug.Print(d.string())
	if v, exists := os.LookupEnv(d.Name); exists {
		if d.Config.CopyValue {
			log.Debug.Printf("Discovered variable %s=%s", d.Name, v)
			return true, map[string]interface{}{d.Name: v}
		}
		log.Debug.Printf("Discovered variable %s", d.Name)
		return true, map[string]interface{}{d.Name: nil}
	}
	return false, nil
}

// DiscoverPrefix  will discover variables by the prefix.
type DiscoverPrefix struct {
	Config DiscoveryConfig
	Prefix string
	// DePrefix if true - will remove the prefix for container.
	// That will enforce Config.CopyValue to true.
	DePrefix bool
}

func (d *DiscoverPrefix) string() string {
	return fmt.Sprintf("DiscoverPrefix{Prefix: %s, DePrefix: %v} with %s", d.Prefix, d.DePrefix, d.Config.string())
}

func (d *DiscoverPrefix) discover(_ host.Host) (bool, map[string]interface{}) {
	log.Debug.Print(d.string())
	m := make(map[string]interface{})
	for _, e := range os.Environ() {
		k := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(k[0], d.Prefix) {
			if d.DePrefix {
				m[strings.TrimPrefix(k[0], d.Prefix)] = k[1]
			} else if d.Config.CopyValue {
				m[k[0]] = k[1]
			} else {
				m[k[0]] = nil
			}
		}
	}
	return len(m) > 0, m
}

// Env represents an environment variables to be defined or passed to the container.
// If the value is nil, variable will be proxied with no explicit value passing.
type Env map[string]interface{}

// AddEnv adds a pair of key:val to the container environment.
// It will be automatically discovered accordingly to configured discovery sources.
// Try each source until something found, if reached the end and nothing found - do nothing.
func (env Env) AddEnv(h host.Host, sources ...Discover) {
	for _, source := range sources {
		found, src := source.discover(h)
		if found {
			log.Debug.Printf("Match found: %s", src)
			for key, val := range src {
				env[key] = val
				log.Debug.Printf("Added variable %s=%s", key, val)
			}
			return
		}
	}
}

// DiscoverEnv use to define RunTainer specific known environment variables
func DiscoverEnv() {
	log.Debug.Print("Discover Environment")

	h := viper.Get("host").(host.Host)

	e := make(Env)
	if en := viper.Get("env"); en != nil {
		log.Debug.Print("Load user defined env settings")
		e = en.(map[string]interface{})
	}

	// just define soma standard host facts as env variables
	e.AddEnv(h, &DiscoverValue{Name: "RT_HOST_NAME", Value: h.Name})
	e.AddEnv(h, &DiscoverValue{Name: "RT_HOST_USER", Value: h.User})
	e.AddEnv(h, &DiscoverValue{Name: "RT_HOST_HOME", Value: h.Home})
	e.AddEnv(h, &DiscoverValue{Name: "RT_HOST_CWD", Value: h.Cwd})

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
			e.AddEnv(h, &DiscoverValue{Name: "DOCKER_HOST", Value: u.String()})
		}
	}

	// now we mirror any env vars that starts with RT_VAR_* and RT_EVAR_* to account for any possible user-defined vars
	e.AddEnv(h, &DiscoverPrefix{Prefix: "RT_VAR_"})
	e.AddEnv(h, &DiscoverPrefix{Prefix: "RT_EVAR_", DePrefix: true})

	log.Debug.Print("Publish to viper")
	viper.Set("env", e)
}
