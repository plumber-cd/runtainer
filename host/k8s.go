package host

import (
	"context"
	"fmt"
	"golang.org/x/term"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
	uexec "k8s.io/client-go/util/exec"
	"k8s.io/kubectl/pkg/cmd/exec"

	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/utils"
)

const serviceAccountNamespace = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

type PodRunMode string

const (
	PodRunModeModeAttach PodRunMode = "PodRunModeModeAttach"
	PodRunModeModeExec   PodRunMode = "PodRunModeModeExec"
	PodRunModeModeLogs   PodRunMode = "PodRunModeModeLogs"
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
	Ports     map[int]int
}

func GetKubeClient() (
	config *rest.Config,
	clientset *kubernetes.Clientset,
	namespace string,
	err error,
) {
	namespace = v1.NamespaceDefault

	if k8sPort := os.Getenv("KUBERNETES_PORT"); k8sPort != "" {
		log.Debug.Printf("Using in-cluster authentication")
		config, err = rest.InClusterConfig()
		if err != nil {
			return
		}

		// It doesn't seem K8s sdk expose any function for detecting current namespace.
		// Best shot I found is here https://github.com/kubernetes/client-go/blob/v0.19.2/tools/clientcmd/client_config.go#L572
		// But `type inClusterClientConfig` is not exported and seems accordingly to the comment only used for internal testing.
		// So best I guess is to mimic same logic here.
		if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
			log.Debug.Printf("Detected POD_NAMESPACE=%s", ns)
			namespace = ns
		} else if data, err := ioutil.ReadFile(serviceAccountNamespace); err == nil {
			log.Debug.Printf("Detected %s", serviceAccountNamespace)
			if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
				log.Debug.Printf("Detected %s=%s", serviceAccountNamespace, ns)
				namespace = ns
			}
		} else {
			log.Debug.Printf("Using NamespaceDefault=%s", namespace)
		}
	} else {
		log.Debug.Printf("Using local kubeconfig")

		clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{},
		)
		config, err = clientConfig.ClientConfig()
		if err != nil {
			return
		}

		namespace, _, err = clientConfig.Namespace()
		if err != nil {
			return
		}
		log.Debug.Printf("Context namespace detected: %s", namespace)
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return
	}

	return
}

func ExecPod(options *PodOptions) error {
	log.Normal.Printf("Running mode: %s", options.Mode)

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

	if options.Mode == PodRunModeModeLogs {
		waitForPod(options.Clientset, pod, v1.PodRunning, v1.PodSucceeded)
		stopEventsWatch.CloseOnce()

		podOptions := &v1.PodLogOptions{
			Container: options.Container,
			Follow:    true,
		}

		req := options.
			Clientset.
			CoreV1().
			Pods(pod.Namespace).
			GetLogs(pod.Name, podOptions)

		podLogs, err := req.Stream(context.TODO())
		if err != nil {
			return err
		}
		go func() {
			if _, err := io.Copy(os.Stdout, podLogs); err != nil {
				log.Normal.Panic(err)
			}
		}()

		return extractExitCode(options.Clientset, pod)
	}

	pod = waitForPod(options.Clientset, pod, v1.PodRunning, v1.PodSucceeded)
	stopEventsWatch.CloseOnce()

	if pod.Status.Phase == v1.PodRunning {
		log.Debug.Printf("Pod is still in the running phase - attempt to establish port forwarding....")
		stopCh := make(chan struct{})
		defer close(stopCh)
		readyCh := make(chan struct{})
		errChan := make(chan error)
		for local, remote := range options.Ports {
			log.Debug.Printf("Forwarding %d:%d", local, remote)

			tunnelReq := options.Clientset.
				CoreV1().
				RESTClient().
				Post().
				Resource("pods").
				Namespace(options.Namespace).
				Name(pod.ObjectMeta.Name).
				SubResource("portforward")

			transport, upgrader, err := spdy.RoundTripperFor(options.Config)
			if err != nil {
				return err
			}

			dialer := spdy.NewDialer(
				upgrader,
				&http.Client{Transport: transport},
				"POST",
				tunnelReq.URL(),
			)

			ports := []string{fmt.Sprintf("%d:%d", local, remote)}
			portforwarder, err := portforward.New(
				dialer,
				ports,
				stopCh,
				readyCh,
				log.Info.Writer(),
				log.Error.Writer(),
			)
			if err != nil {
				return err
			}

			go func() {
				errChan <- portforwarder.ForwardPorts()
			}()

			select {
			case err = <-errChan:
				return err
			case <-portforwarder.Ready:
				log.Debug.Printf("Successfully created port forwarding %d:%d", local, remote)
			}
		}
	}

	var podOptions runtime.Object
	method := "POST"
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

	if err := stream(options, req.URL(), method); err != nil {
		return err
	}

	if options.Mode == PodRunModeModeAttach {
		return extractExitCode(options.Clientset, pod)
	}

	return nil
}

func waitForPod(clientset *kubernetes.Clientset, pod *v1.Pod, phases ...v1.PodPhase) (result *v1.Pod) {
	stop := utils.NewStopChan()

	watchlist := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", pod.Namespace, fields.Everything())
	_, controller := cache.NewInformer(watchlist, &v1.Pod{}, time.Second*1, cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(o, n interface{}) {
			newPod := n.(*v1.Pod)

			// not the pod we created
			if newPod.Name != pod.Name {
				return
			}

			// if the pod is in expected status - stop watching
			for _, phase := range phases {
				if newPod.Status.Phase == phase {
					result = newPod
					stop.CloseOnce()
					return
				}
			}

			if newPod.Status.Phase == v1.PodFailed || newPod.Status.Phase == v1.PodUnknown {
				log.Normal.Printf("Unexpected pod status %s", newPod.Status.Phase)
				stop.CloseOnce()
				return
			}
		},
	})

	controller.Run(stop.Chan)
	return
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

func stream(options *PodOptions, url *url.URL, method string) error {
	streamOptions := remotecommand.StreamOptions{
		Stdin:  options.Stdin,
		Stdout: options.Stdout,
		Stderr: options.Stderr,
		Tty:    options.Tty,
	}

	if options.Stdin != nil {
		switch options.Stdin.(type) {
		case *os.File:
			in := options.Stdin.(*os.File)
			oldState, err := term.MakeRaw(int(in.Fd()))
			if err != nil {
				log.Normal.Panic(err)
			}
			defer func() {
				if err := term.Restore(int(in.Fd()), oldState); err != nil {
					log.Error.Print(err)
				}
			}()
		}
	}

	if streamOptions.Tty {
		s := exec.StreamOptions{
			Namespace:     options.Namespace,
			PodName:       options.PodSpec.ObjectMeta.Name,
			ContainerName: options.Container,
			Stdin:         options.Stdin != nil,
			TTY:           options.Tty,
			IOStreams: genericclioptions.IOStreams{
				In:     options.Stdin,
				Out:    options.Stdout,
				ErrOut: options.Stderr,
			},
		}
		t := s.SetupTTY()
		sizeQueue := t.MonitorSize(t.GetSize())
		streamOptions.TerminalSizeQueue = sizeQueue
		if options.Mode == PodRunModeModeAttach {
			log.Normal.Println("If you don't see a command prompt, try pressing enter.")
		}
	}

	return startStream(method, url, options.Config, streamOptions)
}

func extractExitCode(clientset *kubernetes.Clientset, pod *v1.Pod) error {
	pod = waitForPod(clientset, pod, v1.PodSucceeded, v1.PodFailed)

	unknownRcErr := fmt.Errorf("unknown exit code")

	switch pod.Status.Phase {
	case v1.PodSucceeded:
		return nil
	case v1.PodFailed:
		if len(pod.Status.ContainerStatuses) < 1 || pod.Status.ContainerStatuses[0].State.Terminated == nil {
			return unknownRcErr
		}
		rc := pod.Status.ContainerStatuses[0].State.Terminated.ExitCode
		if rc == 0 {
			return unknownRcErr
		}
		return uexec.CodeExitError{
			Err: fmt.Errorf(
				"terminated (%s)\n%s",
				pod.Status.ContainerStatuses[0].State.Terminated.Reason,
				pod.Status.ContainerStatuses[0].State.Terminated.Message,
			),
			Code: int(rc),
		}
	}

	return unknownRcErr
}
