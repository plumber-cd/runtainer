package volumes

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/plumber-cd/runtainer/host"
	"github.com/plumber-cd/runtainer/image"
	"github.com/spf13/viper"
)

type Volume struct {
	Src  string
	Dest string
}

type Volumes struct {
	ContainerCwd string
	HostMapping  []Volume
}

// DiscoverVolumes discovers volumes
func DiscoverVolumes() {
	volumes := Volumes{}
	if v := viper.Get("volumes"); v != nil {
		mapstructure.Decode(v, &volumes)
	}
	if volumes.HostMapping == nil {
		volumes.HostMapping = make([]Volume, 0)
	}

	h := viper.Get("host").(host.Host)
	i := viper.Get("image").(image.Image)

	volumes.HostMapping = append(volumes.HostMapping, Volume{Src: h.Home, Dest: i.Home + "/rt_host"})

	if strings.HasPrefix(h.Cwd, h.Home) {
		volumes.ContainerCwd = i.Home + "/rt_host" + strings.ReplaceAll(strings.TrimPrefix(h.Cwd, h.Home), "\\", "/")
	} else {
		volumes.ContainerCwd = i.Home + "/rt_cwd/" + filepath.Base(h.Cwd)
		volumes.HostMapping = append(volumes.HostMapping, Volume{Src: h.Cwd, Dest: volumes.ContainerCwd})
	}

	volumes.mirrorInUserHome(h, i, ".local")
	volumes.mirrorInUserHome(h, i, ".cache")
	volumes.mirrorInUserHome(h, i, ".ssh")
	volumes.mirrorInUserHomeVarFileOrDefault(h, i, "AWS_SHARED_CREDENTIALS_FILE", ".aws")
	volumes.mirrorInUserHomeVarOrDefault(h, i, "KUBECONFIG", ".kube")
	volumes.mirrorInUserHomeVarOrDefault(h, i, "GOPATH", "go")
	volumes.mirrorInUserHomeVarOrDefault(h, i, "MAVEN_HOME", ".m2")

	viper.Set("volumes", volumes)
}

func (volumes *Volumes) mirrorInUserHomeVarFileOrDefault(h host.Host, i image.Image, v string, path string) {
	if s := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); s != "" {
		volumes.mirrorInUserHomePathOrDefault(h, i, filepath.Dir(s), path)
	} else {
		volumes.mirrorInUserHome(h, i, path)
	}
}

func (volumes *Volumes) mirrorInUserHomeVarOrDefault(h host.Host, i image.Image, v string, path string) {
	volumes.mirrorInUserHomePathOrDefault(h, i, os.Getenv(v), path)
}

func (volumes *Volumes) mirrorInUserHomePathOrDefault(h host.Host, i image.Image, s string, path string) {
	if isDirExists(s) {
		volumes.HostMapping = append(volumes.HostMapping, Volume{Src: s, Dest: i.Home + "/" + path})
	} else {
		volumes.mirrorInUserHome(h, i, path)
	}
}

func (volumes *Volumes) mirrorInUserHome(h host.Host, i image.Image, path string) {
	local := h.Home + "/" + path
	if isDirExists(local) {
		volumes.HostMapping = append(volumes.HostMapping, Volume{Src: local, Dest: i.Home + "/" + path})
	}
}

func isDirExists(path string) bool {
	src, err := os.Stat(path)
	if err != nil {
		return false
	}

	return src.IsDir()
}
