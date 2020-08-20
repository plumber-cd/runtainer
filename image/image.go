package image

import (
	"os/exec"

	"github.com/plumber-cd/runtainer/host"
	"github.com/spf13/viper"
)

// Image holds facts about the image
type Image struct {
	Name          string
	OS            string
	PathSeparator string
	User          string
	Home          string
}

// DiscoverImage discover facts about the image
func DiscoverImage(image string) {
	// TODO: for now we assume all containers are Linux
	os := "linux"
	pathSeparator := "/"
	whoamiCmd := Cmd(image, "whoami")
	user := host.Exec(whoamiCmd)
	homeCmd := Cmd(image, "(cd && pwd)")
	home := host.Exec(homeCmd)

	viper.Set("image", Image{
		Name:          image,
		OS:            os,
		PathSeparator: pathSeparator,
		User:          user,
		Home:          home,
	})
}

// Cmd constructs a command that can be host.Exec (or else).
// Useful to run something in the container.
// Note the current implementation always uses `docker run --rm` regardless what backend is configured.
// Essentially that will span up new container every call which might be costly.
// This behavior may and certainly will change in future towards
// improving execution speed and using actually configured backend.
func Cmd(image string, cmd string) *exec.Cmd {
	return exec.Command(viper.Get("host").(host.Host).DockerPath, "run", "--rm", "--entrypoint", "sh", image, "-c", cmd)
}
