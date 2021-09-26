package host

import (
	"context"
	"io"
	"net/url"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/cmd/exec"

	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/utils"
)

type PodOptions struct {
	Config    *rest.Config
	Clientset *kubernetes.Clientset
	Namespace string
	PodSpec   *v1.Pod
	Container string
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	Tty       bool
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

	podOptions := v1.PodAttachOptions{
		Container: options.Container,
		Stdin:     options.Stdin != nil,
		Stdout:    options.Stdout != nil,
		Stderr:    options.Stderr != nil,
		TTY:       options.Tty,
	}
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
			Stdin:         podOptions.Stdin,
			TTY:           podOptions.TTY,
		}
		t := s.SetupTTY()
		sizeQueue := t.MonitorSize(t.GetSize())
		streamOptions.TerminalSizeQueue = sizeQueue
		log.Stderr.Println("If you don't see a command prompt, try pressing enter.")
	}

	req := options.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("attach")
	req.VersionedParams(
		&podOptions,
		scheme.ParameterCodec,
	)

	return startStream("POST", req.URL(), options.Config, streamOptions)
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
