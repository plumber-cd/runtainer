package host

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/user"
	"sync"
	"time"

	"github.com/docker/cli/cli/streams"
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
				log.Normal.Printf("Failed cleaning up pod %s: %s", pod.ObjectMeta.Name, err)
			}
		}
	}()

	stopEventsWatch := watchPodEvents(options.Clientset, pod)
	waitForPod(options.Clientset, pod)
	stopEventsWatch.CloseOnce()

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
				log.Normal.Printf("Unexpected pod status %s", newPod.Status.Phase)
				stop.CloseOnce()
				return
			}
		},
	})

	controller.Run(stop.Chan)
}

func watchPodEvents(clientset *kubernetes.Clientset, pod *v1.Pod) *utils.StopChan {
	stop := utils.NewStopChan()
	mutex := sync.Mutex{}

	watchlist := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "events", pod.Namespace,
		fields.Everything())
	_, controller := cache.NewInformer(
		watchlist,
		&v1.Event{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				mutex.Lock()
				defer mutex.Unlock()

				e := obj.(*v1.Event)

				if e.InvolvedObject.Kind != "Pod" {
					return
				}
				if e.InvolvedObject.Namespace != pod.Namespace {
					return
				}
				if e.InvolvedObject.Name != pod.Name {
					return
				}

				log.Normal.Printf(
					"k8s event [%s] [%s]: %s",
					e.Type,
					e.Reason,
					e.Message,
				)
			},
		},
	)

	go controller.Run(stop.Chan)
	return stop
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

	if options.Stdin != nil {
		switch options.Stdin.(type) {
		case *streams.In:
			in := options.Stdin.(*streams.In)
			if err := in.SetRawTerminal(); err != nil {
				log.Normal.Panic(err)
			}
			defer in.RestoreTerminal()
		}
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
		log.Normal.Println("If you don't see a command prompt, try pressing enter.")
	}

	return startStream("POST", url, options.Config, streamOptions)
}
