package cmd

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/plumber-cd/runtainer/host"
	"github.com/plumber-cd/runtainer/image"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/volumes"
	"github.com/spf13/viper"
)

func runInDocker(args []string) {
	h := viper.Get("host").(host.Host)
	i := viper.Get("image").(image.Image)
	v := viper.Get("volumes").(volumes.Volumes)

	dockerArgs, inDockerArgs := splitArgs(args...)

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
		dockerExecArgs = append(dockerExecArgs, "--env", env.Name+"="+env.Value)
	}
	for _, vol := range v.HostMapping {
		dockerExecArgs = append(dockerExecArgs, "--volume", vol.Src+":"+vol.Dest)
	}
	dockerExecArgs = append(dockerExecArgs, "--workdir", v.ContainerCwd)
	if runtime.GOOS != "windows" {
		dockerExecArgs = append(dockerExecArgs, "--group-add", h.GID)
	}
	dockerExecArgs = append(dockerExecArgs, dockerArgs...)
	dockerExecArgs = append(dockerExecArgs, i.Name)
	dockerExecArgs = append(dockerExecArgs, inDockerArgs...)
	log.Debug.Printf("dockerExecArgs: %s", strings.Join(dockerExecArgs, " "))

	dockerExecCommand := exec.Command(h.DockerPath, dockerExecArgs...)
	log.Info.Printf("Executing docker: %s", dockerExecCommand.String())

	dockerExecCommand.Stdin = os.Stdin
	dockerExecCommand.Stdout = os.Stdout
	dockerExecCommand.Stderr = os.Stderr

	if err := dockerExecCommand.Run(); err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			log.Error.Print(err)
			os.Exit(exitErr.ExitCode())
		}

		log.Error.Panic(err)
	}
}
