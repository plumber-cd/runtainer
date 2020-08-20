package host

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/mapstructure"
	"github.com/plumber-cd/runtainer/log"
	"github.com/spf13/viper"
)

// EnvVar represents an environment variable to be defined on the container
// If the value is nil, variable will be proxied with no explicit value passing.
type EnvVar struct {
	Name  string
	Value interface{}
}

// Host facts about the host
type Host struct {
	Name        string
	User        string
	UID         string
	GID         string
	Home        string
	Cwd         string
	DockerPath  string
	KubectlPath string
	Env         []EnvVar
}

// DiscoverHost discover information about the host
func DiscoverHost() {
	hostName, err := os.Hostname()
	if err != nil {
		log.Error.Panic(err)
	}

	currentUser, err := user.Current()
	if err != nil {
		log.Error.Panic(err)
	}

	home, err := homedir.Dir()
	if err != nil {
		log.Error.Panic(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Error.Panic(err)
	}

	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		log.Error.Panic(err)
	}

	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil && viper.GetBool("kube") {
		log.Error.Panic(err)
	}

	h := Host{}
	if hst := viper.Get("host"); hst != nil {
		// when read from viper for the first time (i.e. nothing Set it there yet as the struct) it will be a map[string]interface{}
		// hence we need to convert it to the struct
		err := mapstructure.Decode(hst, &h)
		if err != nil {
			log.Error.Panic(err)
		}
	}

	h.Name = hostName
	h.User = currentUser.Username
	h.UID = currentUser.Uid
	h.GID = currentUser.Gid
	h.Home = home

	// What to assume a host cwd when executing container
	if d := viper.GetString("dir"); d != "" {
		h.Cwd, err = filepath.Abs(d)
		if err != nil {
			log.Error.Panic(err)
		}
	} else {
		h.Cwd = cwd
	}

	h.DockerPath = dockerPath
	h.KubectlPath = kubectlPath

	if h.Env == nil {
		h.Env = make([]EnvVar, 0)
	}

	// publish the host facts to the viper
	viper.Set("host", h)

	// once host facts are discovered, we may proceed discovering environment variables
	discoverEnv()
}

// Exec exec command on the host and return the output
func Exec(cmd *exec.Cmd) string {
	log.Info.Printf("Executing: %s", cmd.String())

	out, err := cmd.Output()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			log.Error.Print(string(exitErr.Stderr))
		}
		log.Error.Panic(err)
	}

	return strings.TrimSpace(string(out))
}
