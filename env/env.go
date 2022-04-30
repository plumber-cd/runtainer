package env

import (
	"fmt"
	"os"
	"strconv"
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

// Port represents container ports mapping
type Ports map[int]int

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
	if en := viper.Get("environment"); en != nil {
		log.Debug.Print("Load user defined environment settings")
		e = en.(map[string]interface{})
	}

	// just define soma standard host facts as env variables
	e.AddEnv(h, &DiscoverValue{Name: "RT_HOST_NAME", Value: h.Name})
	e.AddEnv(h, &DiscoverValue{Name: "RT_HOST_USER", Value: h.User})
	e.AddEnv(h, &DiscoverValue{Name: "RT_HOST_HOME", Value: h.Home})
	e.AddEnv(h, &DiscoverValue{Name: "RT_HOST_CWD", Value: h.Cwd})

	// now we mirror any env vars that starts with RT_VAR_* and RT_EVAR_* to account for any possible user-defined vars
	e.AddEnv(h, &DiscoverPrefix{Prefix: "RT_VAR_"})
	e.AddEnv(h, &DiscoverPrefix{Prefix: "RT_EVAR_", DePrefix: true})

	for _, v := range viper.GetStringSlice("env") {
		log.Debug.Printf("Parsing --env=%s", v)
		split := strings.SplitN(v, "=", 2)
		switch len(split) {
		case 1:
			e.AddEnv(h, &DiscoverVariable{Name: split[0]})
		case 2:
			e.AddEnv(h, &DiscoverValue{Name: split[0], Value: split[1]})
		default:
			log.Normal.Fatalf("Invalid input for --env=%s", v)
		}
	}

	log.Debug.Print("Publish to viper")
	viper.Set("environment", e)
}

// DiscoverPorts
func DiscoverPorts() {
	log.Debug.Print("Discover Ports")

	p := make(Ports)
	if en := viper.Get("ports"); en != nil {
		log.Debug.Print("Load user defined ports settings")
		p = en.(map[int]int)
	}

	for _, port := range viper.GetStringSlice("port") {
		log.Debug.Printf("Parsing --port=%s", port)
		portSplit := strings.Split(port, ":")
		if len(portSplit) != 2 {
			log.Normal.Fatalf("Invalid input for --port=%s", port)
		}
		local, err := strconv.Atoi(portSplit[0])
		if err != nil {
			log.Normal.Fatal(err)
		}
		remote, err := strconv.Atoi(portSplit[1])
		if err != nil {
			log.Normal.Fatal(err)
		}
		p[local] = remote
	}

	log.Debug.Print("Publish to viper")
	viper.Set("ports", p)
}
