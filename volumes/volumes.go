package volumes

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/plumber-cd/runtainer/host"
	"github.com/plumber-cd/runtainer/image"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/utils"
	"github.com/spf13/viper"
)

const (
	rtHostHome = "rt_host_home"
	rtCwd      = "rt_cwd"
)

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
			log.Stderr.Panic(err)
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

	// now we will determine current working directory inside
	if strings.HasPrefix(h.Cwd, h.Home) {
		log.Debug.Printf("Host cwd %s is under user home %s, calculating container cwd accordingly", h.Cwd, h.Home)
		// basically, if current working directory on the host somewhere under the user home, we already have it mounted - we just need to calculate the path to it
		containerRtHomePath, err := filepath.Rel(h.Home, h.Cwd)
		if err != nil {
			log.Stderr.Panic(err)
		}
		// just to get rid of . and ..
		containerRtHomePath, err = filepath.Abs(containerRtHomePath)
		if err != nil {
			log.Stderr.Panic(err)
		}
		// convert path separator to what's in the image
		// note that filepath.FromSlash and filepath.ToSlash won't work as they would rely on the host OS file separator
		switch i.PathSeparator {
		case "\\":
			containerRtHomePath = strings.ReplaceAll(containerRtHomePath, "/", "\\")
		case "/":
			containerRtHomePath = strings.ReplaceAll(containerRtHomePath, "\\", "/")
		default:
			log.Stderr.Fatalf("Unknown path separator: %s", i.PathSeparator)
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

// Resolve ~ into user home.
// This is platform agnostic - always uses slash as a separator.
func resolveTilde(h, p string) string {
	// resolve ~ (ir present) to the actual user home
	if strings.HasPrefix(p, "~") {
		p = strings.TrimPrefix(p, "~")
		p = path.Join(h, p)
	}

	return p
}

// AddHostMount adds a pair of src:dest to the mounts.
// If src didn't existed - does nothing.
// Only for mounting directories.
// Automatically resolves ~ to the user home (both host and container).
func (volumes *Volumes) AddHostMount(h host.Host, i image.Image, src, dest string) {
	log.Debug.Printf("Checking potential volume %s:%s", src, dest)

	src = resolveTilde(h.Home, src)
	dest = resolveTilde(i.Home, dest)

	// just in case - get rid of .. and etc
	// do that only for src as filepath uses host file separator
	s, err := filepath.Abs(src)
	if err != nil {
		log.Stderr.Panic(err)
	}

	exists, err := utils.OsFs.DirExists(s)
	if err != nil {
		log.Stderr.Panic(err)
	}

	if exists {
		log.Debug.Printf("Adding a valid volume %s:%s", src, dest)
		volumes.HostMapping = append(volumes.HostMapping, Volume{Src: src, Dest: dest})
	}
}

// AddMirrorHostMount basically falls back to addHostMount, using p as both source and dest.
// Useful for mirroring user home folders like .aws, .ssh and etc.
func (volumes *Volumes) AddMirrorHostMount(h host.Host, i image.Image, p string) {
	volumes.AddHostMount(h, i, p, p)
}

// AddEnvVarToDirMountOrDefault look if environment variable v is defined,
// if yes - use it's value as src and path as dest and call addHostMount;
// otherwise - use path to call addMirrorHostMount.
// Useful to mount folders that might be defined as env variables like MAVEN_HOME, but if not always has default hardcoded location.
func (volumes *Volumes) AddEnvVarToDirMountOrDefault(h host.Host, i image.Image, v string, path string) {
	p, e := os.LookupEnv(v)
	if e {
		volumes.AddHostMount(h, i, p, path)
	} else {
		volumes.AddMirrorHostMount(h, i, path)
	}
}

// AddEnvVarToDirMountOrExecOrDefault look if environment variable v is defined and use AddEnvVarToDirMountOrDefault in that case,
// if not try to read exec output assuming it's an equivalent for the env value;
// otherwise - use path to call addMirrorHostMount.
// Useful to mount folders that might be defined as env variables like GOPATH or GOCACHE, or by a command like `go env`,
// and if not always has default hardcoded location.
func (volumes *Volumes) AddEnvVarToDirMountOrExecOrDefault(h host.Host, i image.Image, v string, ex []string, path string) {
	// we might use `go env` to determine some values later
	b, bErr := exec.LookPath(ex[0])

	if _, exists := os.LookupEnv(v); exists {
		volumes.AddEnvVarToDirMountOrDefault(h, i, v, path)
	} else if bErr == nil {
		p := host.Exec(exec.Command(b, ex[1:]...))
		volumes.AddHostMount(h, i, p, path)
	} else {
		volumes.AddMirrorHostMount(h, i, path)
	}
}

// AddEnvVarToFileMountOrDefault it basically mimics addEnvVarToDirMountOrDefault, except that environment variable treated as a path to file.
// We only mount directories, so parent directory to the file will be determined and used for mounting via addHostMount.
// Useful for mounting folders that may be defined as env variables by a path to the file,
// such as AWS_SHARED_CREDENTIALS_FILE and KUBECONFIG, but if not always has default hardcoded location.
func (volumes *Volumes) AddEnvVarToFileMountOrDefault(h host.Host, i image.Image, v string, path string) {
	p, e := os.LookupEnv(v)
	if e {
		volumes.AddHostMount(h, i, filepath.Dir(p), path)
	} else {
		volumes.AddMirrorHostMount(h, i, path)
	}
}
