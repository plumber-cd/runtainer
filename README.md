# runtainer

Run anything as a container (in local Kubernetes cluster).

- [runtainer](#runtainer)
  - [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
      - [Mac](#mac)
      - [Windows Native](#windows-native)
      - [Windows WSL2](#windows-wsl2)
      - [Linux](#linux)
    - [Installation](#installation)
      - [Brew](#brew)
      - [From Release](#from-release)
      - [From Sources](#from-sources)
    - [Usage](#usage)
      - [Basic container run](#basic-container-run)
      - [Exec into container and stay there](#exec-into-container-and-stay-there)
      - [Extra volumes and port forwarding](#extra-volumes-and-port-forwarding)
      - [Extra ENV variables](#extra-env-variables)
      - [Custom directory](#custom-directory)
      - [Piping](#piping)
      - [Troubleshooting](#troubleshooting)
    - [Configuration](#configuration)
  - [Why](#why)
  - [Disclaimer](#disclaimer)
  - [How](#how)

## Getting Started

### Prerequisites

You should be running local Kubernetes cluster. Your Kube Config (`~/.kube/config` or `KUBECONFIG`) should be pointing to that local cluster. RT will use your default context and a namespace set in it.

RT will run just fine if you point it to a remote cluster - but it would make a very little sense since environment variables and local paths would not exist on the remote cluster node. Most likely the pod will just fail to start.

Status for some of the possible setups that were tested:

#### Mac

- Docker for Desktop: :white_check_mark:
- Lima+K3s: :white_check_mark:
- Rancher Desktop: :x: (it's the same as Lima+K3s under the hood - but see https://github.com/rancher-sandbox/rancher-desktop/issues/678)

#### Windows Native

- Docker for Desktop: :x: (it would work but Docker VM uses `/mnt/host` instead of just `/mnt` - RT will have to figure out detecting that automatically - it can be fixed)
- WSL2+K3s: :x: (seems like k3s required systemd which is missing in Ubuntu WSL2)
- Rancher Desktop: :white_check_mark:

#### Windows WSL2

- Docker for Desktop: :white_check_mark: (only in `/mnt` - Docker VM can't mount other VM's paths; alternative - move your code to the Docker VM)
- WSL2+K3s: :x: (seems like k3s required systemd which is missing in Ubuntu WSL2)
- Rancher Desktop: :white_check_mark: (only in `/mnt` - Rancher VM can't mount other VM's paths; alternative - move your code to the Rancher VM)

#### Linux

- Docker CE: :grey_question:
- ContainerD: :grey_question:
- K3s: :grey_question:

### Installation

#### Brew

Installation with `brew` is coming later.

#### From Release

Download a binary from [Releases](https://github.com/plumber-cd/runtainer/releases). All binaries are built with GitHub Actions and you can inspect [how](.github/workflows/release.yml).

Don't forget to add it to your `PATH`.

#### From Sources

You can build it yourself, of course (and Go made it really easy):

```bash
go install github.com/plumber-cd/runtainer@${version}
```

Don't forget to add it to your `PATH`.

### Usage

See full CLI [docs](./runtainer.md).

Image name serves as a separator. To its left RT's own arguments are expected. To it's right - command and arguments to be passed to the container.

Command separated from the arguments by a `--`, as described in the POSIX chapter 12.02, Guideline 10: https://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap12.html#tag_12_02. Both command and arguments are two arrays of strings in Kubernetes API, so from the user perspective - separator is not required to be used explicitly as K8s will accept it just fine without it. You will want to use a separator when you need to use default `ENTRYPOINT` (`.spec.container[].command`) from the image while passing custom arguments `CMD` (`.spec.container[].args`) to it.

There are 3 modes RT can run an image:

1. By default, preferred method is `PodRunModeModeExec`. RT will use `.spec.container[].command = ["cat"]` without arguments to start a pod (which will keep it running doing nothing indefinitely) and then it will exec into that pod executing combined command and arguments from the RT input. Technically, in this mode there is no difference between interactive and non-interactive mode, so `--interactive` flag is effectively ignored.

        runtainer alpine sh # interactive
        runtainer alpine whoami # non-interactive

   This mode is preferred because RT can guarantee nothing will get executed in the container that the user can miss in the terminal and because this mode does not have downsides of other modes.

2. When there is no command passed to the RT (even if arguments were passed) - RT will not be able to determine default `ENTRYPOINT` of the image to run the previous mode. In this case RT will use `PodRunModeModeAttach` mode - it will set `.spec.container[].args` accordingly to RT input and start a pod and then attach to it.

        runtainer alpine -- sh

   In this mode there's always a chance that the user may miss some activity in the container from the moment the pod became `running` and RT was able to attach to it.
   Probably due to that (or might be some other weird bug in `k8s.io/client-go`) - you almost 100% guaranteed not going to see initial command prompt after the attachment, that's why you will be presented with a message `If you don't see a command prompt, try pressing enter.` - the same one `kubectl run` will present you with.
   Also, this mode requires user to explicitly tell RT if that is interactive session or not. Otherwise if it is not interactive - by the time RT will try to attach the pod might already be in `succeeded` state, so RT will fail with an ugly message. You can try it: `runtainer alpine -- whoami`.
   Lastly, regardless of `--stdin` value - RT will not be able to pass its own StdIn to the container. The main container process has been already started by kubelet without RT participation and RT attaches to it mid-term. Even though any input after the attachment will be transferred to the container just fine - any initial StdIn such as pipe from another program will not be possible:

        echo hi | runtainer alpine -- xargs echo # will not print hi

3. Third mode `PodRunModeModeLogs` essentially is complimentary to `PodRunModeModeAttach` - it's basically the same one but it's expecting pod to become `succeeded` without user interaction so it will not ever try to attach to it. Instead it will use pod logs API to stream logs which can be done as long as the pod exists (even after it succeeded before it's being garbage-collected).

        runtainer --interactive=false alpine -- whoami

   One big downside of this mode is that it essentially disables both `--stdin` and `--tty` - all it does is just streaming logs in read-only mode.
   It also disables `--port` for the same reason.

With that - a simple rule is to try to use default `PodRunModeModeExec` mode as much as possible, unless `ENTRYPOINT` of the image is not known - then one of the `PodRunModeModeAttach` or `PodRunModeModeLogs` might help.

Below is a few more examples:

#### Basic container run

```bash
runtainer alpine whoami
runtainer maven:3.6.3-jdk-14 mvn -version
runtainer maven:3.6.3-jdk-14 mvn -- -version
```

#### Exec into container and stay there

```bash
runtainer alpine sh
```

#### Extra volumes and port forwarding

This one is especially fun as basically this Jenkins Master in the container will have aws/k8s/etc access right from your laptop. How cool is that, huh?

```bash
JENKINS_VERSION=2.289.1-lts
runtainer -v $(cd && pwd)/.jenkins:/var/jenkins_home -p 8080:8080 jenkins/jenkins:${JENKINS_VERSION}
```

Or maybe you want to test a Jenkins plugin?

```bash
runtainer -p 8080:8080 maven:3.8.1-jdk-8 mvn -Dhost=0.0.0.0 clean hpi:run
```

#### Extra ENV variables

```bash
runtainer -e PROFILE maven:3.6.3-jdk-14 mvn -version
# OR
runtainer -e PROFILE=foo maven:3.6.3-jdk-14 mvn -version
```

It will also automatically discover ENV variables with `RT_VAR_*` and `RT_EVAR_*` prefixes.

```bash
RT_VAR_FOO=foo RT_EVAR_BAR=bar runtainer alpine env
```

You will see that `RT_VAR_FOO` was passed to the container as-is, and `RT_EVAR_BAR` was passed to the container as `BAR` i.e. removing the prefix.

#### Custom directory

```bash
runtainer -d /tmp alpine touch hi.txt
```

#### Piping

```bash
echo hi | runtainer -t=false alpine xargs echo
```

By default `runtainer` run containers with `--stdin` and `--tty`, so you could use container interactively with minimum extra moves.
But for piping we need to explicitly disable `--tty` so it could read stdin from a pipe and not a virtual terminal.
You can even pipe multiple `runtainer` calls too:

```bash
runtainer alpine echo hi | runtainer -t=false alpine xargs echo
```

#### Troubleshooting

Use `--log` to make it write additional diag messages to a log file in the current working directory. Use `--debug` to write even more verbose diag messages.

### Configuration

The tool can be configured with both files and ENV variables.

ENV variables prefixed with `RT_` may be used, i.e. `RT_LOG=true`.

For config files it supports many config file formats, including JSON, TOML, YAML, HCL, envfile and Java properties formats.
Log files can be located in multiple locations, and they read on top of each other with merging in the following order:

1. Global system-level config in `~/.runtainer/config` (can be overridden with `--config`)
1. Local config file `.runtainer.yaml` (or any other supported extension) in the directory `runtainer` is started in
1. Local config file `.runtainer.yaml` (or any other supported extension) in the directory `runtainer` specified by `--dir` (if any)

This can be useful to configure on per-project or per-directory basis with committing that config to Git.

Config files and env variables apart from being an alternative source for CLI args configuration, can also be used to configure various host, image or container related things. See the following yaml example:

```yaml
log: true
environment:
  # Var FOO have an explicit hardcoded value here
  FOO: foo
  # Var BAR does not have a value, so it will be mirrored from the host at runtime
  BAR: null
volumes:
  hostMapping:
    - src: /home/foo/bar
      dest: /root/baz
```

Run it with `--debug` and you will be able to visually inspect what the tool automatically discovers for you.

## Why

Container's purpose is to run every process in its own isolated, self-contained and portable environment.
This is a very well known and widely adopted concept in application development.
What is less commonly used is the fact that you can also pack your tools as a container image and run it anywhere. Think of container images like a `brew` or `chocolatey` or even `pyenv`. You never actually have to install anything on your workstation anymore. You can easily choose which version of what you want to use by just changing the image tag.

Consider the following example.
You need to run AWS CLI.
You either pack it as a container image yourself or use some existing image, like [`amazon/aws-cli`](https://hub.docker.com/r/amazon/aws-cli).
You can do something like this:

```bash
docker run amazon/aws-cli sts get-caller-identity
```

Of course, isolated container will have no AWS credentials whatsoever, but if you have an active session on your workstation, you can do something like:

```bash
docker run -v $(cd && pwd)/.aws:/root/.aws -e AWS_PROFILE -e AWS_DEFAULT_REGION amazon/aws-cli sts get-caller-identity
```

You can go very far with this concept. You can mount other similar known locations (let's say `~/.m2`, `~/.kube` etc.), current working directory, and pass any known environment variables to the inside. Basically your workstation considered stateful and used as a persistent storage, while various containers can easily reproduce any runtime environment you need (any os/tools/etc).

You can also easily choose which version of your tool you want to run at any given time:

```bash
docker run amazon/aws-cli:2.0.40 sts get-caller-identity
```

And in case you are using multiple versions all the time, which is common especially in a micro-stack world for stability purposes, you don't need to re-install the software all the time. So, containers also can be used as `pyenv`, `virtualenv` or `rvm` of some sort.

Also, part of the image could be any number of useful wrappers and scripts, so you can package complex routines involving multiple tools into one executable artifact and distribute it to your team, so they can run it 1:1 as you did.

As you probably already imagined, the only problem with this approach is - a lot of typing. You got to provide so many arguments to mount all the locations and pass in all the environment variables. This is a lot of routine work and it is boring.

This tool basically aims to remove that overhead. It will automatically discover all known file system locations and environment variables and pass them to the container so you don't have to. It is basically a smart equivalent for `docker run` or `kubernetes run` commands.

For portability under the hood RT is using Kubernetes. That way no matter your container engine choice - the tool will always be able to run.

## Disclaimer

Be mindful of what you run.
This tool's mere purpose is to run the container as if its internals were installed on your host.
That means the usual isolation attributed to the containers in general is compromised, any potential malicious programs could easily escape to the host level and mess with your stuff.
You should really know and trust containers you run with RunTainer as like you were to do it with regular software you install on your desktop.

## How

This tool will do the following:

1. Discover and mount to your container well known locations (such as `~/.ssh`, `~/.aws` etc) as well as user-defined locations
1. Expose to your container well known environment variables (such as `AWS_PROFILE`, `AWS_DEFAULT_REGION` etc) as well as user-defined environment variables
1. Mount `cwd` and make it a `cwd` inside the container
1. Make UID/GID to match your host inside the container
1. Run the command you specified in your container (or default `ENTRYPOINT`)
