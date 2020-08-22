package docker

import (
	"os/exec"
	"runtime"

	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/host"
	"github.com/plumber-cd/runtainer/log"
	"github.com/spf13/viper"
)

// Run this will use all discovered facts and user input to run container using Docker CLI as a backend engine.
func Run(dockerArgs, inDockerArgs []string) {
	log.Debug.Print("Starting docker backend")

	h, i, v := discover.GetFromViper()

	dockerExecArgs := make([]string, 0)
	dockerExecArgs = append(dockerExecArgs, "run")

	dockerExecArgs = append(dockerExecArgs, "--rm")
	dockerExecArgs = append(dockerExecArgs, "--attach", "STDOUT")
	dockerExecArgs = append(dockerExecArgs, "--attach", "STDERR")
	if viper.GetBool("stdin") {
		log.Debug.Print("--stdin mode enabled")
		dockerExecArgs = append(dockerExecArgs, "--interactive")
		dockerExecArgs = append(dockerExecArgs, "--attach", "STDIN")
	}
	if viper.GetBool("tty") {
		log.Debug.Print("--tty mode enabled")
		dockerExecArgs = append(dockerExecArgs, "--tty")
	}
	for _, env := range h.Env {
		val := env.Name
		if env.Value != nil {
			val = val + "=" + env.Value.(string)
		}
		log.Debug.Printf("Adding env variable: %s", val)
		dockerExecArgs = append(dockerExecArgs, "--env", val)
	}
	for _, vol := range v.HostMapping {
		log.Debug.Printf("Adding volume: %s:%s", vol.Src, vol.Dest)
		dockerExecArgs = append(dockerExecArgs, "--volume", vol.Src+":"+vol.Dest)
	}
	log.Debug.Printf("Using cwd: %s", v.ContainerCwd)
	dockerExecArgs = append(dockerExecArgs, "--workdir", v.ContainerCwd)
	if runtime.GOOS != "windows" {
		log.Debug.Printf("Since the platform is %s, add GID %s", runtime.GOOS, h.GID)
		dockerExecArgs = append(dockerExecArgs, "--group-add", h.GID)
	}
	dockerExecArgs = append(dockerExecArgs, dockerArgs...)
	dockerExecArgs = append(dockerExecArgs, i.Name)
	dockerExecArgs = append(dockerExecArgs, inDockerArgs...)

	dockerExecCommand := exec.Command(h.DockerPath, dockerExecArgs...)
	host.ExecBackend(dockerExecCommand)
}
