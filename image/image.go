package image

import (
	"os/exec"

	"github.com/plumber-cd/runtainer/host"
	"github.com/spf13/viper"
)

// Image facts about the image
type Image struct {
	Name          string
	OS            string
	PathSeparator string
	User          string
	Home          string
}

// DiscoverImage discover information about the image
func DiscoverImage(image string) {
	// TODO: for now we assume all containers are Linux
	os := "linux"
	pathSeparator := "/"
	whoamiCmd := cmdDocker(image, "whoami")
	user, _ := host.Exec(whoamiCmd)
	homeCmd := cmdDocker(image, "(cd && pwd)")
	home, _ := host.Exec(homeCmd)

	viper.Set("image", Image{
		Name:          image,
		OS:            os,
		PathSeparator: pathSeparator,
		User:          user,
		Home:          home,
	})
}

func cmdDocker(image string, cmd string) *exec.Cmd {
	return exec.Command(viper.Get("host").(host.Host).DockerPath, "run", "--rm", "--entrypoint", "sh", image, "-c", cmd)
}
