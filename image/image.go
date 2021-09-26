package image

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/plumber-cd/runtainer/host"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/utils"
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
	log.Debug.Print("Discover image")

	kubeconfig, clientset, err := host.GetKubeClient()
	if err != nil {
		log.Normal.Panic(err)
	}

	// TODO: should come from viper
	namespace := "default"
	podName := fmt.Sprintf("runtainer-%s", utils.RandomHex(4))
	containerName := "runtainer"

	containerSpec := v1.Container{
		Name:            containerName,
		Image:           image,
		Command:         []string{"cat"},
		TTY:             true,
		ImagePullPolicy: v1.PullPolicy(v1.PullIfNotPresent),
	}
	podSpec := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{containerSpec},
		},
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	podOptions := host.PodOptions{
		Config:    kubeconfig,
		Clientset: clientset,
		Namespace: namespace,
		PodSpec:   &podSpec,
		Container: containerName,
		Mode:      host.PodRunModeModeExec,
		ExecCmd: []string{
			"/bin/sh",
			"-c",
			"echo $(whoami):$(cd && pwd)",
		},
		Stdout: stdout,
		Stderr: stderr,
	}

	if viper.GetBool("dry-run") {
		log.Debug.Print("--dry-run mode enabled")
		s := kjson.NewYAMLSerializer(kjson.DefaultMetaFactory, scheme.Scheme,
			scheme.Scheme)
		err = s.Encode(&podSpec, os.Stderr)
		if err != nil {
			log.Normal.Panic(err)
		}
	}

	if err := host.ExecPod(&podOptions); err != nil {
		log.Normal.Println(stderr.String())
		log.Normal.Panic(err)
	}

	out := strings.TrimSpace(stdout.String())
	outSplit := strings.Split(out, ":")
	if len(outSplit) != 2 {
		log.Normal.Println(stderr.String())
		log.Normal.Panic(fmt.Errorf("Unexpected output: %s", out))
	}
	username := outSplit[0]
	pwd := outSplit[1]

	// TODO: for now we assume all containers are Linux
	os := "linux"
	pathSeparator := "/"

	log.Debug.Print("Publish to viper")
	viper.Set("image", Image{
		Name:          image,
		OS:            os,
		PathSeparator: pathSeparator,
		User:          username,
		Home:          pwd,
	})
}
