package host

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/user"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/cmd/exec"

	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/utils"
)

type PodRunMode string

const (
	PodRunModeModeAttach PodRunMode = "PodRunModeModeAttach"
	PodRunModeModeExec   PodRunMode = "PodRunModeModeExec"
)

type PodOptions struct {
	Config    *rest.Config
	Clientset *kubernetes.Clientset
	Namespace string
	PodSpec   *v1.Pod
	Container string
	Mode      PodRunMode
	ExecCmd   []string
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	Tty       bool
}

func GetKubeClient() (*rest.Config, *kubernetes.Clientset, error) {
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

func ExecPod(options *PodOptions) error {
	podsClient := options.Clientset.CoreV1().Pods(options.Namespace)

	pod, err := podsClient.Create(context.TODO(), options.PodSpec, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err := podsClient.Delete(context.TODO(), pod.ObjectMeta.Name, metav1.DeleteOptions{}); err != nil {
			if err != nil {
				log.Stderr.Printf("Failed cleaning up pod %s: %s", pod.ObjectMeta.Name, err)
			}
		}
	}()

	waitForPod(options.Clientset, pod)

	var podOptions runtime.Object
	req := options.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace)

	switch options.Mode {
	case PodRunModeModeAttach:
		podOptions = &v1.PodAttachOptions{
			Container: options.Container,
			Stdin:     options.Stdin != nil,
			Stdout:    options.Stdout != nil,
			Stderr:    options.Stderr != nil,
			TTY:       options.Tty,
		}

		req = req.SubResource("attach")
	case PodRunModeModeExec:
		podOptions = &v1.PodExecOptions{
			Container: options.Container,
			Command:   options.ExecCmd,
			Stdin:     options.Stdin != nil,
			Stdout:    options.Stdout != nil,
			Stderr:    options.Stderr != nil,
			TTY:       options.Tty,
		}

		req = req.SubResource("exec")
	default:
		return fmt.Errorf("Unknown run mode: %s", options.Mode)
	}

	req.VersionedParams(
		podOptions,
		scheme.ParameterCodec,
	)

	return stream(options, req.URL())
}

func waitForPod(clientset *kubernetes.Clientset, pod *v1.Pod) {
	stop := utils.NewStopChan()

	watchlist := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", pod.Namespace, fields.Everything())
	_, controller := cache.NewInformer(watchlist, &v1.Pod{}, time.Second*1, cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(o, n interface{}) {
			newPod := n.(*v1.Pod)

			// not the pod we created
			if newPod.Name != pod.Name {
				return
			}

			// if the pod is running, stop watching and continue with the cmd execution
			if newPod.Status.Phase == v1.PodRunning {
				stop.CloseOnce()
				return
			}

			if newPod.Status.Phase != v1.PodPending {
				log.Stderr.Printf("Unexpected pod status %s", newPod.Status.Phase)
				stop.CloseOnce()
				return
			}
		},
	})

	controller.Run(stop.Chan)
}

func startStream(
	method string,
	url *url.URL,
	config *restclient.Config,
	streamOptions remotecommand.StreamOptions,
) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}

	return exec.Stream(streamOptions)
}

func stream(options *PodOptions, url *url.URL) error {
	streamOptions := remotecommand.StreamOptions{
		Stdin:  options.Stdin,
		Stdout: options.Stdout,
		Stderr: options.Stderr,
		Tty:    options.Tty,
	}

	if streamOptions.Tty {
		s := exec.StreamOptions{
			Namespace:     options.Namespace,
			PodName:       options.PodSpec.ObjectMeta.Name,
			ContainerName: options.Container,
			Stdin:         options.Stdin != nil,
			TTY:           options.Tty,
		}
		t := s.SetupTTY()
		sizeQueue := t.MonitorSize(t.GetSize())
		streamOptions.TerminalSizeQueue = sizeQueue
		log.Stderr.Println("If you don't see a command prompt, try pressing enter.")
	}

	return startStream("POST", url, options.Config, streamOptions)
}