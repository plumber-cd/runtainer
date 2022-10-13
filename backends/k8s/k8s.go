package k8s

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/moby/term"
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

func ptr[T any](v T) *T {
	return &v
}

func Run(containerCmd, containerArgs []string) {
	log.Debug.Print("Starting k8s backend")

	stdIn, stdOut, stdErr := term.StdStreams()

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
		EnvFrom:         []v1.EnvFromSource{},
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
		Stdout:    stdOut,
		Stderr:    stdErr,
	}

	if secret := viper.GetString("secret"); secret != "" {
		log.Debug.Print("--secret enabled")
		podSpec.Spec.ImagePullSecrets = []v1.LocalObjectReference{
			{Name: secret},
		}
	}

	if viper.GetBool("run-as-current-user") {
		podSpec.Spec.SecurityContext.RunAsUser = &h.UID
	}

	podSpec.Spec.SecurityContext.SupplementalGroups = []int64{h.GID}

	if viper.GetBool("run-as-current-user") && viper.GetBool("run-as-current-group") {
		podSpec.Spec.SecurityContext.RunAsGroup = &h.GID
	} else if h.GID > 0 {
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

	for _, secret := range viper.GetStringSlice("secrets.env") {
		cfg := strings.Split(secret, ":")
		secret = cfg[0]
		log.Info.Printf("Adding env envFrom: %s", secret)

		envFromSource := v1.EnvFromSource{
			SecretRef: &v1.SecretEnvSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: secret,
				},
				Optional: ptr(true),
			},
		}

		for _, option := range cfg {
			if strings.HasPrefix(option, "prefix=") {
				envFromSource.Prefix = strings.SplitN(option, "=", 2)[1]
				log.Info.Printf("Secret envFrom %s: custom prefix %s", secret, envFromSource.Prefix)
			}
		}

		containerSpec.EnvFrom = append(containerSpec.EnvFrom, envFromSource)
	}

	for _, vol := range v.HostMapping {
		volumeName := fmt.Sprintf("runtainer-%s", utils.RandomHex(4))
		src := vol.Src
		dst := vol.Dest

		if runtime.GOOS == "windows" {
			log.Debug.Printf("Since the platform is %s, convert local disks to /mnt", runtime.GOOS)
			split := strings.SplitN(src, ":\\", 2)
			if len(split) != 2 {
				log.Normal.Fatal(fmt.Errorf("Failed to convert windows path %s", src))
			}
			src = fmt.Sprintf("/mnt/%s/%s", strings.ToLower(split[0]), split[1])
			src = strings.Replace(src, "\\", "/", -1)
		}

		log.Info.Printf("Adding volume %s: %s:%s", volumeName, src, dst)
		podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: src,
				},
			},
		})
		containerSpec.VolumeMounts = append(containerSpec.VolumeMounts, v1.VolumeMount{
			Name:      volumeName,
			MountPath: dst,
		})
	}

	for _, secret := range viper.GetStringSlice("secrets.volumes") {
		cfg := strings.Split(secret, ":")
		secret = cfg[0]
		dst := "/rt-secrets/" + secret
		log.Info.Printf("Adding secret volume %s -> %s", secret, dst)

		volume := v1.Volume{
			Name: secret,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName:  secret,
					DefaultMode: ptr(int32(0600)), // since we use fsGroup - it will result in 0640 in reality
					Optional:    ptr(true),
				},
			},
		}
		volumeMount := v1.VolumeMount{
			Name:      secret,
			MountPath: dst,
			ReadOnly:  true,
		}

		for _, option := range cfg {
			if strings.HasPrefix(option, "mountPath=") {
				volumeMount.MountPath = strings.SplitN(option, "=", 2)[1]
				log.Info.Printf("Secret volume %s: custom mountPath %s", secret, volumeMount.MountPath)
			} else if strings.HasPrefix(option, "item=") {
				item := strings.SplitN(option, "=", 2)[1]
				keyToPath := v1.KeyToPath{
					Key:  item,
					Path: item,
				}
				volume.VolumeSource.Secret.Items = append(volume.VolumeSource.Secret.Items, keyToPath)
				log.Info.Printf("Secret volume %s: custom items %v", secret, volume.VolumeSource.Secret.Items)
			}
		}

		podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, volume)
		containerSpec.VolumeMounts = append(containerSpec.VolumeMounts, volumeMount)
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
		podOptions.Stdin = stdIn
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
