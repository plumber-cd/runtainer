package image

import (
	"bytes"
	"fmt"
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

	kubeconfig, clientset, namespace, err := host.GetKubeClient()
	if err != nil {
		log.Normal.Panic(err)
	}

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

	if secret := viper.GetString("secret"); secret != "" {
		log.Debug.Print("--secret enabled")
		podSpec.Spec.ImagePullSecrets = []v1.LocalObjectReference{
			{Name: secret},
		}
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

	imageProbeBuf := new(bytes.Buffer)
	kubeJsonSerializer := kjson.NewYAMLSerializer(kjson.DefaultMetaFactory, scheme.Scheme,
		scheme.Scheme)
	if err := kubeJsonSerializer.Encode(&podSpec, imageProbeBuf); err != nil {
		log.Normal.Panic(err)
	}
	log.Debug.Printf("Image probe pod: %s", imageProbeBuf.String())

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
