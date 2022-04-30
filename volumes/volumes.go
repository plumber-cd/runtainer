package volumes

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/plumber-cd/runtainer/host"
	"github.com/plumber-cd/runtainer/image"
	"github.com/plumber-cd/runtainer/log"
	"github.com/spf13/viper"
)

const (
	rtHostHome = "rt_host_home"
	rtCwd      = "rt_cwd"
)

// DiscoveryConfig various configuration for discovery
type DiscoveryConfig struct {
	UseParent bool
}

func (dc *DiscoveryConfig) string() string {
	return fmt.Sprintf("DiscoveryConfig{UseParent: %v}", dc.UseParent)
}

// Discover is an interface for various discoverers
type Discover interface {
	// Discover must return true only if discovered, validated and ensured it is usable
	discover(h host.Host, i image.Image, dest string) (found bool, src string)
}

// DiscoverMirror basically is a default fallback discoverer.
// It will use mounting dest and try it as a source on the host.
// Useful for mirroring user home folders like .aws, .ssh and etc.
type DiscoverMirror struct {
	Config DiscoveryConfig
}

func (d *DiscoverMirror) string() string {
	return fmt.Sprintf("DiscoverMirror{} with %s", d.Config.string())
}

func (d *DiscoverMirror) discover(h host.Host, _ image.Image, dest string) (bool, string) {
	log.Debug.Printf("%s, dest: %s", d.string(), dest)
	return checkLocalDir(h, d.Config, dest)
}

// DiscoverDir uses Path specified to discover dir.
// Useful for mounting directories that we know or can calculate specific path to.
type DiscoverDir struct {
	Config DiscoveryConfig
	Path   string
}

func (d *DiscoverDir) string() string {
	return fmt.Sprintf("DiscoverDir{Path: %s} with %s", d.Path, d.Config.string())
}

func (d *DiscoverDir) discover(h host.Host, _ image.Image, dest string) (bool, string) {
	log.Debug.Printf("%s", d.string())
	return checkLocalDir(h, d.Config, d.Path)
}

// DiscoverEnvVar look for environment variable.
// Useful to mount folders that might be defined as env variables like MAVEN_HOME.
type DiscoverEnvVar struct {
	Config DiscoveryConfig
	EnvVar string
}

func (d *DiscoverEnvVar) string() string {
	return fmt.Sprintf("DiscoverEnvVar{EnvVar: %s} with %s", d.EnvVar, d.Config.string())
}

func (d *DiscoverEnvVar) discover(h host.Host, _ image.Image, _ string) (bool, string) {
	log.Debug.Printf("%s", d.string())
	src, exists := os.LookupEnv(d.EnvVar)
	if !exists {
		log.Debug.Printf("%s variable was not found", d.EnvVar)
		return false, ""
	}
	log.Debug.Printf("Found %s=%s", d.EnvVar, src)
	return checkLocalDir(h, d.Config, src)
}

// DiscoverExec will execute a command on the host and consider it's output as a source.
// Useful to mount folders that might be defined by a command like `go env`.
type DiscoverExec struct {
	Config DiscoveryConfig
	Args   []string
}

func (d *DiscoverExec) string() string {
	return fmt.Sprintf("DiscoverExec{Args: [%s]} with %s", strings.Join(d.Args, ", "), d.Config.string())
}

func (d *DiscoverExec) discover(h host.Host, _ image.Image, _ string) (bool, string) {
	log.Debug.Printf("%s", d.string())
	bin, err := exec.LookPath(d.Args[0])
	if bin == "" || err != nil {
		log.Debug.Printf("%s binary was not found (%s)", d.Args[0], err)
		return false, ""
	}
	log.Debug.Printf("Found binary %s", bin)
	src := host.Exec(exec.Command(bin, d.Args[1:]...))
	return checkLocalDir(h, d.Config, src)
}

// DiscoverCallback will execute a callback function.
// Useful to mount folders that might be defined by a some other Go code, such as https://github.com/helm/helm/blob/v3.3.0/pkg/helmpath/lazypath_windows.go.
type DiscoverCallback struct {
	Config   DiscoveryConfig
	Callback func(h host.Host, i image.Image, dest string) (bool, string)
}

func (d *DiscoverCallback) string() string {
	return fmt.Sprintf("DiscoverCallback{Callback: <...>} with %s", d.Config.string())
}

func (d *DiscoverCallback) discover(h host.Host, i image.Image, dest string) (bool, string) {
	log.Debug.Printf("%s, dest: %s", d.string(), dest)
	exists, src := d.Callback(h, i, dest)
	if !exists {
		log.Debug.Printf("Callback did not found any source (%s)", src)
		return exists, src
	}
	log.Debug.Printf("Callback returned %s", src)
	return checkLocalDir(h, d.Config, src)
}

// Volume struct contains a pair of source and destination paths for mounting
type Volume struct {
	Src  string
	Dest string
}

// Volumes struct is an extendable info as to what to mount into container
type Volumes struct {
	// ContainerCwd current working directory to be set inside the container
	ContainerCwd string
	// HostMapping list of volumes to mount from the host
	HostMapping []Volume
}

// AddHostMount adds a pair of src:dest to the mounts.
// Src will be automatically discovered accordingly to configured discovery sources.
// Try each source until something found, if reached the end and nothing found - do nothing.
// Only for mounting directories.
// Automatically resolves ~ to the user home (both host and container).
func (v *Volumes) AddHostMount(h host.Host, i image.Image, dest string, sources ...Discover) {
	log.Debug.Printf("Discovering potential volume mount for %s", dest)
	for _, source := range sources {
		found, src := source.discover(h, i, dest)
		if found {
			log.Debug.Printf("Match found %s:%s", src, dest)
			if dest == "" {
				dest = src
			}
			dest = resolveTilde(i.Home, dest)
			v.HostMapping = append(v.HostMapping, Volume{Src: src, Dest: dest})
			log.Debug.Printf("Added volume %s:%s", src, dest)
			return
		}
	}
	log.Debug.Printf("Nothing found for volume mount %s", dest)
}

// DiscoverVolumes analyze environment to determine what to mount
func DiscoverVolumes() {
	log.Debug.Print("Discover volumes")

	volumes := Volumes{}

	// read user defined volumes to mount
	if v := viper.Get("volumes"); v != nil {
		log.Debug.Print("Load user defined volumes settings")
		// when read from viper for the first time (i.e. nothing Set it there yet as the struct) it will be a map[string]interface{}
		// hence we need to convert it to the struct
		err := mapstructure.Decode(v, &volumes)
		if err != nil {
			log.Normal.Panic(err)
		}
	}

	// fix structure in case nothing was defined
	if volumes.HostMapping == nil {
		volumes.HostMapping = make([]Volume, 0)
	}

	// Host and image should have been already analyzed by this point
	h := viper.Get("host").(host.Host)
	i := viper.Get("image").(image.Image)

	// first of all, add mount for the host home
	// use path and not filepath as host path separator is irrelevant to what's inside the container
	hostHomeMount := path.Join(i.Home, rtHostHome)
	volumes.HostMapping = append(volumes.HostMapping, Volume{
		Src:  h.Home,
		Dest: hostHomeMount,
	})

	for _, vol := range viper.GetStringSlice("volume") {
		log.Debug.Printf("Parsing --volume=%s", vol)
		volSplit := strings.Split(vol, ":")
		if len(volSplit) != 2 {
			log.Normal.Fatalf("Invalid input for --volume=%s", vol)
		}
		volumes.HostMapping = append(volumes.HostMapping, Volume{
			Src:  volSplit[0],
			Dest: volSplit[1],
		})
	}

	// now we will determine current working directory inside
	if strings.HasPrefix(h.Cwd, h.Home) {
		log.Debug.Printf("Host cwd %s is under user home %s, calculating container cwd accordingly", h.Cwd, h.Home)
		// basically, if current working directory on the host somewhere under the user home, we already have it mounted - we just need to calculate the path to it
		containerRtHomePath, err := filepath.Rel(h.Home, h.Cwd)
		if err != nil {
			log.Normal.Panic(err)
		}
		// convert path separator to what's in the image
		// note that filepath.FromSlash and filepath.ToSlash won't work as they would rely on the host OS file separator
		switch i.PathSeparator {
		case "\\":
			containerRtHomePath = strings.ReplaceAll(containerRtHomePath, "/", "\\")
		case "/":
			containerRtHomePath = strings.ReplaceAll(containerRtHomePath, "\\", "/")
		default:
			log.Normal.Fatalf("Unknown path separator: %s", i.PathSeparator)
		}
		// again, this is for the container so host path separator is irrelevant, hence path not filepath
		volumes.ContainerCwd = path.Join(hostHomeMount, containerRtHomePath)
	} else {
		log.Debug.Printf("Host cwd %s seems to be outside user home %s, calculating and mounting container cwd accordingly", h.Cwd, h.Home)
		// otherwise, we need to mount host home separately
		// we start off with a host cwd base name, as some software cares about cwd name (I'm looking at you, Helm)
		containerRtHomePath := filepath.Base(h.Cwd)
		// now calculate full path to cwd inside
		volumes.ContainerCwd = path.Join(i.Home, rtCwd, containerRtHomePath)
		// and add it to mounts
		volumes.HostMapping = append(volumes.HostMapping, Volume{Src: h.Cwd, Dest: volumes.ContainerCwd})
	}

	log.Debug.Print("Publish to viper")
	viper.Set("volumes", volumes)
}
