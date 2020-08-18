package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/plumber-cd/runtainer/host"
	"github.com/plumber-cd/runtainer/image"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/volumes"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

func runInKube(args []string) {
	h := viper.Get("host").(host.Host)
	i := viper.Get("image").(image.Image)
	v := viper.Get("volumes").(volumes.Volumes)

	suffix := randomHex(4)
	podName := "runtainer-" + suffix

	kubeArgs, containerArgs := splitArgs(args...)

	kubectlExecArgs := make([]string, 0)
	kubectlExecArgs = append(kubectlExecArgs, podName)
	kubectlExecArgs = append(kubectlExecArgs, "--image="+i.Name)
	kubectlExecArgs = append(kubectlExecArgs, "--image-pull-policy=IfNotPresent")
	kubectlExecArgs = append(kubectlExecArgs, "--generator=run-pod/v1")
	kubectlExecArgs = append(kubectlExecArgs, "--labels=runtainer=true,runtainer_name="+podName)
	kubectlExecArgs = append(kubectlExecArgs, "--quiet")
	kubectlExecArgs = append(kubectlExecArgs, "--restart=Never")

	for _, env := range h.Env {
		kubectlExecArgs = append(kubectlExecArgs, "--env", env.Name+"="+env.Value)
	}

	kubectlDryRunExecArgs := []string{"run", "--dry-run", "-o", "yaml"}
	kubectlDryRunExecArgs = append(kubectlDryRunExecArgs, kubectlExecArgs...)
	kubectlDryRunExecArgs = append(kubectlDryRunExecArgs, kubeArgs...)
	kubectlDryRunExecArgs = append(kubectlDryRunExecArgs, "--attach=false")
	kubectlDryRunExecArgs = append(kubectlDryRunExecArgs, "--tty=false")
	kubectlDryRunExecArgs = append(kubectlDryRunExecArgs, "--stdin=false")
	kubectlDryRunExecArgs = append(kubectlDryRunExecArgs, "--rm=false")
	kubectlDryRunExecArgs = append(kubectlDryRunExecArgs, "--")
	kubectlDryRunExecArgs = append(kubectlDryRunExecArgs, containerArgs...)
	kubectlDryRunExecCommand := exec.Command(h.KubectlPath, kubectlDryRunExecArgs...)
	kubectlDryRunExecYaml, kubectlDryRunExecOutput := host.Exec(kubectlDryRunExecCommand)
	log.Debug.Print(kubectlDryRunExecYaml)

	var pod, service map[interface{}]interface{}
	dec := yaml.NewDecoder(bytes.NewReader(kubectlDryRunExecOutput))
	for {
		var doc map[interface{}]interface{}
		if dec.Decode(&doc) != nil {
			break
		}
		switch doc["kind"] {
		case "Pod":
			pod = doc
		case "Service":
			service = doc
		}
	}

	volumes := getFromMap(&pod, "spec.volumes")
	if volumes == nil {
		setInMap(&pod, "spec.volumes", make([]map[interface{}]interface{}, 0))
	}
	// containers := getFromMap(&pod, "spec.containers")
	// container := (*containers).([]map[interface{}]interface{})[0]
	// volumeMounts := getFromMap(&pod, "spec.volumes")
	if volumes == nil {
		setInMap(&pod, "spec.volumes", make([]map[interface{}]interface{}, 0))
	}
	for _, vol := range v.HostMapping {
		log.Error.Print(vol)
		log.Error.Print(*getFromMap(&pod, "spec.containers"))

		// dockerExecArgs = append(dockerExecArgs, "--volume", vol.Src+":"+vol.Dest)
	}

	log.Error.Print(pod)
	log.Error.Print(service)
	log.Error.Fatal("Done")

	kubectlExecArgs = append(kubectlExecArgs, "--rm")
	kubectlExecArgs = append(kubectlExecArgs, "--attach", "STDOUT")
	kubectlExecArgs = append(kubectlExecArgs, "--attach", "STDERR")
	if viper.GetBool("stdin") {
		log.Debug.Print("--stdin mode enabled")
		kubectlExecArgs = append(kubectlExecArgs, "--interactive")
		kubectlExecArgs = append(kubectlExecArgs, "--attach", "STDIN")
	}
	if viper.GetBool("tty") {
		log.Debug.Print("--tty mode enabled")
		kubectlExecArgs = append(kubectlExecArgs, "--tty")
	}
	kubectlExecArgs = append(kubectlExecArgs, "--workdir", v.ContainerCwd)
	if runtime.GOOS != "windows" {
		kubectlExecArgs = append(kubectlExecArgs, "--group-add", h.GID)
	}
	kubectlExecArgs = append(kubectlExecArgs, kubeArgs...)
	kubectlExecArgs = append(kubectlExecArgs, i.Name)
	kubectlExecArgs = append(kubectlExecArgs, containerArgs...)
	log.Debug.Printf("dockerExecArgs: %s", strings.Join(kubectlExecArgs, " "))

	dockerExecCommand := exec.Command(h.DockerPath, kubectlExecArgs...)
	log.Info.Printf("Executing docker: %s", dockerExecCommand.String())

	dockerExecCommand.Stdin = os.Stdin
	dockerExecCommand.Stdout = os.Stdout
	dockerExecCommand.Stderr = os.Stderr

	if err := dockerExecCommand.Run(); err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			log.Error.Print(err)
			os.Exit(exitErr.ExitCode())
		}

		log.Error.Panic(err)
	}
}

func getFromMap(m *map[interface{}]interface{}, k string) *interface{} {
	address := strings.Split(k, ".")
	r := (*m)[address[0]]
	if len(address) == 1 {
		if r == nil {
			return nil
		}
		return &r
	}
	x := r.(map[interface{}]interface{})
	return getFromMap(&x, strings.Join(address[1:], "."))
}

func setInMap(m *map[interface{}]interface{}, k string, v interface{}) {
	address := strings.Split(k, ".")
	if len(address) == 1 {
		(*m)[address[0]] = v
		return
	}
	r := (*m)[address[0]]
	if r == nil {
		(*m)[address[0]] = make(map[interface{}]interface{})
		r = (*m)[address[0]]
	}
	x := r.(map[interface{}]interface{})
	setInMap(&x, strings.Join(address[1:], "."), v)
}
