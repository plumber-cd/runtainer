# runtainer

Run anything as a container

- [runtainer](#runtainer)
  - [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
      - [Mac](#mac)
      - [Windows](#windows)
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

You should be running local Kubernetes cluster. Your Kube Config (`~/.kube/config` or `KUBECONFIG`) should be pointing to that local cluster. The tool will run just fine if you point it to remote cluster - but it would make little sense since environment variables and local paths would not exist on the remote cluster node.

It has been tested in the following scenarios:

#### Mac

- Docker for Desktop: :white_check_mark:
- Lima+K3s: :white_check_mark:
- Rancher Desktop: :x: (it's the same as Lima+K3s under the hood - but see https://github.com/rancher-sandbox/rancher-desktop/issues/678)

#### Windows

- WSL2+K3s: :grey_question:
- Rancher Desktop: :grey_question:

#### Linux

- Docker CE: :grey_question:
- ContainerD: :grey_question:

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

Basically the idea is that on the left of the image - you provide any arguments to the `runtainer` itself. On the right (all the way up to the `--` separator) is a CMD that will be passed as-is to the container. The rest after the `--` separator - will be supplied as args to that container.

A few examples are below.

#### Basic container run

```bash
runtainer alpine whoami
runtainer maven:3.6.3-jdk-14 -- -version
runtainer maven:3.6.3-jdk-14 mvn -version
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
runtainer -p 8080:8080 maven:3.8.1-jdk-8 -- -Dhost=0.0.0.0 clean hpi:run
```

#### Extra ENV variables

```bash
runtainer -e PROFILE maven:3.6.3-jdk-14 -- -version
# OR
runtainer -e PROFILE=foo maven:3.6.3-jdk-14 -- -version
```

It will also automatically discover ENV variables with `RT_VAR_*` and `RT_EVAR_*` prefixes.

```bash
RT_VAR_FOO=foo RT_EVAR_BAR=bar runtainer alpine env
```

You will see that `RT_VAR_FOO` was passed to the container as-is, and `RT_EVAR_BAR` was passed to the container as `BAR` i.e. removing the prefix.

#### Custom directory

```bash
runtainer -d /tmp alpine -- touch hi.txt
```

#### Piping

```bash
echo hi | runtainer -t=false alpine -- xargs echo
runtainer alpine -- echo hi | runtainer -t=false alpine -- xargs echo
```

By default `runtainer` runs containers with `--stdin` and `--tty`, so you could use container interactively with minimum extra moves.
Note we had to explicitly disable `--tty` so it could read stdin from a pipe.
You can even pipe multiple `runtainer` calls too.

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
env:
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

Containers purpose is to run every process in its own isolated, self-contained and portable environment.
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

For portability under the hood it is using Kubernetes. That way no matter your container engine choice - the tool will always be able to run.

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
