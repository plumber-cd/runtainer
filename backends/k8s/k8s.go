package k8s

import (
	"errors"
	"fmt"
	"os"
	"os/user"

	"github.com/docker/cli/cli/streams"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/host"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/utils"
	"github.com/spf13/viper"
)

func getClient() (*rest.Config, *kubernetes.Clientset, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, nil, err
	}

	var config *rest.Config

	if k8sPort := os.Getenv("KUBERNETES_PORT"); k8sPort != "" {
		log.Debug.Printf("Using in-cluster authentication")
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, err
		}
	} else {
		log.Debug.Printf("Using local kubeconfig")
		var kubeconfig string

		if cfg := os.Getenv("KUBECONFIG"); cfg != "" {
			kubeconfig = cfg
		} else {
			home := usr.HomeDir
			if home == "" {
				return nil, nil, errors.New("home directory unknown")
			}
			kubeconfig = fmt.Sprintf("%s/.kube/config", home)
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	return config, clientset, nil
}

func Run(containerCmd, containerArgs []string) {
	log.Debug.Print("Starting k8s backend")

	h, e, i, v := discover.GetFromViper()

	kubeconfig, clientset, err := getClient()
	if err != nil {
		log.Stderr.Panic(err)
	}

	// TODO: should come from viper
	namespace := "default"
	podName := fmt.Sprintf("runtainer-%s", utils.RandomHex(4))
	containerName := "runtainer"

	containerSpec := v1.Container{
		Name:            containerName,
		Image:           i.Name,
		Command:         containerCmd,
		Args:            containerArgs,
		ImagePullPolicy: v1.PullPolicy(v1.PullIfNotPresent),
		Env:             []v1.EnvVar{},
		VolumeMounts:    []v1.VolumeMount{},
		SecurityContext: &v1.SecurityContext{
			Privileged: utils.BoolPtr(true),
		},
	}
	podSpec := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: v1.PodSpec{
			Volumes: []v1.Volume{},
			SecurityContext: &v1.PodSecurityContext{
				SupplementalGroups: []int64{h.GID},
				FSGroup:            &h.GID,
			},
		},
	}

	for key, val := range e {
		var str string
		if val == nil {
			str = os.Getenv(key)
		} else {
			str = val.(string)
		}
		log.Info.Printf("Adding env variable: %s=%s", key, str)
		containerSpec.Env = append(containerSpec.Env, v1.EnvVar{
			Name:  key,
			Value: str,
		})
	}

	for _, vol := range v.HostMapping {
		volumeName := fmt.Sprintf("runtainer-%s", utils.RandomHex(4))
		log.Info.Printf("Adding volume %s: %s:%s", volumeName, vol.Src, vol.Dest)
		podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: vol.Src,
				},
			},
		})
		containerSpec.VolumeMounts = append(containerSpec.VolumeMounts, v1.VolumeMount{
			Name:      volumeName,
			MountPath: vol.Dest,
		})
	}

	if viper.GetBool("stdin") {
		log.Debug.Print("--tty mode enabled for container")
		containerSpec.Stdin = true
	}

	if viper.GetBool("tty") {
		log.Debug.Print("--tty mode enabled for container")
		containerSpec.TTY = true
	}

	podSpec.Spec.Containers = []v1.Container{containerSpec}

	if viper.GetBool("dry-run") {
		log.Debug.Print("--dry-run mode enabled")
		s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme,
			scheme.Scheme)
		err = s.Encode(&podSpec, os.Stderr)
		if err != nil {
			log.Stderr.Panic(err)
		}
		return
	}

	podOptions := host.PodOptions{
		Config:    kubeconfig,
		Clientset: clientset,
		Namespace: namespace,
		PodSpec:   &podSpec,
		Container: containerName,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	}

	if viper.GetBool("stdin") {
		log.Debug.Print("--stdin mode enabled")

		in := streams.NewIn(os.Stdin)
		if err := in.SetRawTerminal(); err != nil {
			log.Stderr.Panic(err)
		}
		defer in.RestoreTerminal()

		podOptions.Stdin = in
	}

	if viper.GetBool("tty") {
		log.Debug.Print("--tty mode enabled")
		podOptions.Tty = true
	}

	log.Info.Printf("Using cwd: %s", v.ContainerCwd)

	if err := host.ExecPod(&podOptions); err != nil {
		log.Stderr.Panic(err)
	}
}
