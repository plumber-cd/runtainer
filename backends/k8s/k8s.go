package k8s

import (
	"bytes"
	"fmt"
	"os"

	"github.com/docker/cli/cli/streams"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/exec"

	"github.com/plumber-cd/runtainer/discover"
	"github.com/plumber-cd/runtainer/host"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/utils"
	"github.com/spf13/viper"
)

func Run(containerCmd, containerArgs []string) {
	log.Debug.Print("Starting k8s backend")

	h, e, p, i, v := discover.GetFromViper()

	kubeconfig, clientset, namespace, err := host.GetKubeClient()
	if err != nil {
		log.Normal.Panic(err)
	}

	podName := fmt.Sprintf("runtainer-%s", utils.RandomHex(4))
	containerName := "runtainer"

	log.Info.Printf("Using cwd: %s", v.ContainerCwd)

	containerSpec := v1.Container{
		Name:            containerName,
		Image:           i.Name,
		Command:         containerCmd,
		Args:            containerArgs,
		WorkingDir:      v.ContainerCwd,
		ImagePullPolicy: v1.PullPolicy(v1.PullIfNotPresent),
		Env:             []v1.EnvVar{},
		VolumeMounts:    []v1.VolumeMount{},
	}
	podSpec := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: v1.PodSpec{
			Volumes:         []v1.Volume{},
			SecurityContext: &v1.PodSecurityContext{},
			RestartPolicy:   v1.RestartPolicyNever,
		},
	}
	podOptions := host.PodOptions{
		Config:    kubeconfig,
		Clientset: clientset,
		Namespace: namespace,
		PodSpec:   &podSpec,
		Container: containerName,
		Mode:      host.PodRunModeModeAttach,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	}

	if h.GID > 0 {
		podSpec.Spec.SecurityContext.SupplementalGroups = []int64{h.GID}
		podSpec.Spec.SecurityContext.FSGroup = &h.GID
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

	podOptions.Ports = p

	if len(containerSpec.Command) > 0 {
		podOptions.Mode = host.PodRunModeModeExec
		podOptions.ExecCmd = append(containerSpec.Command, containerSpec.Args...)
		containerSpec.Command = []string{"cat"}
		containerSpec.Args = []string{}
	} else {
		if viper.GetBool("interactive") {
			log.Debug.Print("--interactive mode enabled")
			podOptions.Mode = host.PodRunModeModeAttach
		} else {
			log.Debug.Print("--interactive mode disabled")
			podOptions.Mode = host.PodRunModeModeLogs
		}
	}

	if viper.GetBool("stdin") {
		log.Debug.Print("--stdin mode enabled")
		containerSpec.Stdin = true
		podOptions.Stdin = streams.NewIn(os.Stdin)
	}

	if viper.GetBool("tty") {
		log.Debug.Print("--tty mode enabled")
		containerSpec.TTY = true
		podOptions.Tty = true
	}

	podSpec.Spec.Containers = []v1.Container{containerSpec}

	podSpecJsonBuf := new(bytes.Buffer)
	kubeJsonSerializer := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme,
		scheme.Scheme)
	if err := kubeJsonSerializer.Encode(&podSpec, podSpecJsonBuf); err != nil {
		log.Normal.Panic(err)
	}
	podSpecJson := podSpecJsonBuf.String()
	log.Debug.Printf("Pod: %s", podSpecJson)
	if viper.GetBool("dry-run") {
		log.Debug.Print("--dry-run mode enabled")
		fmt.Println(podSpecJson)
		return
	}

	if err := host.ExecPod(&podOptions); err != nil {
		switch e := err.(type) {
		case exec.CodeExitError:
			os.Exit(e.ExitStatus())
		default:
			log.Normal.Panic(err)
		}
	}
}
