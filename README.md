# runtainer

Run anything as a container

- [runtainer](#runtainer)
  - [Getting Started](#getting-started)
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

### Installation

#### Brew

Installation with `brew` is coming later.

#### From Release

Download a binary from [Releases](https://github.com/plumber-cd/runtainer/releases). All binaries built with GitHub Actions and you can inspect [how](.github/workflows/release.yml).

Don't forget to add it to your `PATH`.

#### From Sources

You can build it yourself, of course (and Go made it really easy):

```bash
go install github.com/plumber-cd/runtainer@${version}
```

Don't forget to add it to your `PATH`.

### Usage

See full CLI [docs](./runtainer.md).

Basically the idea is that on the left of the image you provide any arguments to the `runtainer` itself, on the right all the way before `--` args will be passed as-is to container engine i.e. `docker`, and the rest will be supplied as `CMD` to the container.

A few examples are below.

#### Basic container run

```bash
runtainer alpine -- whoami
runtainer maven:3.6.3-jdk-14 -- mvn -version
```

#### Exec into container and stay there

```bash
runtainer alpine -- sh
# OR
runtainer alpine --entrypoint sh
```

Of course `--entrypoint` is a regular `--docker` arg.

#### Extra volumes and port forwarding

This one is especially fun as basically this Jenkins Master in the container will have aws/k8s/etc access from your laptop. How cool is that huh?

```bash
runtainer jenkins/jenkins:2.235.5-lts -v $(cd && pwd)/.jenkins:/var/jenkins_home -p 8080:8080
```

`-v` and `-p` are `docker` args as well.

#### Extra ENV variables

```bash
runtainer maven:3.6.3-jdk-14 -e PROFILE -- mvn -version
# OR
runtainer maven:3.6.3-jdk-14 -e PROFILE=foo -- mvn -version
```

`-e` is a `docker` arg.

It will also discover ENV variables with `RT_VAR_*` and `RT_EVAR_*` prefixes.

```bash
RT_VAR_FOO=foo RT_EVAR_BAR=bar runtainer alpine -- env
```

You will see that `RT_VAR_FOO` was passed to the container as-is, and `RT_EVAR_BAR` was passed to the container as `BAR` i.e. removing the prefix.

#### Custom directory

```bash
runtainer -d /tmp alpine -- touch hi.txt
```

Here `-d` on the left was an arg of `runtainer` itself.

#### Piping

```bash
echo hi | runtainer -t=false alpine -- xargs echo
runtainer alpine -- echo hi | runtainer -t=false alpine -- xargs echo
```

By default `runtainer` runs containers with `--interactive` and `--tty`, so you could use container interactively with minimum extra moves.
Note we had to explicitly disable `--tty` so it could read from the host stdout.
You can even pipe multiple `runtainer` calls too.

#### Troubleshooting

By default, `runtainer` designed so it doesn't leave any residue as it's whole idea is to run internals of the container like they were installed right on the host.
It's designed to be transparent.
Most and foremost it is important to STDOUT and STDERR, we can't afford `runtainer` to pollute the output from the container as it might be meant to be consumed by something else (like `json`/`yaml` output).
Log files also viewed as potentially unwanted leftovers.
With that in mind, it leaves no traces of it's own unless it fails, then and only then it prints error message and/or stack trace to STDERR.
It might not be enough for troubleshooting so there is two args `--log` and `--verbose`. The later automatically enables the first. It spits out `runtainer.log` file to the directory it is started from.

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
host:
  env:
    # Var FOO have an explicit hardcoded value here
    - name: FOO
      value: Bar
    # Var BAR does not have a value, so it will be mirrored from the host at runtime
    - name: BAR
volumes:
  hostMapping:
    - src: /home/foo/bar
      dest: /root/baz
```

Run it with `--verbose` and you will be able to visually inspect what the tool automatically discovers for you.

## Why

Containers purpose is to run every process in it's own isolated, self-contained and portable environment.
This is a very well known and widely adopted concept in application development.
What is less commonly used is the fact that you can also pack your toolset as a container image and run it anywhere. Think containers like a `brew` or `chocolatey`, except that you never actually install anything on your workstation, so as a bonus you can easily choose which version of what you want to use.

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

You can go very far with this concept. You can mount other similar known locations (let's say `~/.m2`, `~/.kube` etc.), current working directory, and pass any known environment variables to the inside. Basically your workstation considered stateful and used as a persistent storage, and with containers you can easily reproduce any runtime environment you need (any platform/os/toolset/etc).

You can also easily choose which version of your toolset you want to run at any given time:

```bash
docker run amazon/aws-cli:2.0.40 sts get-caller-identity
```

And in case you are using multiple versions all the time, which is common especially in a micro-stack world for stability purposes, you don't need to re-install the software all the time. So, containers also can be used as `virtualenv` or `rvm` of some sort.

Also, part of the image could be any number of useful wrappers and scripts, so you can package complex routines involving multiple tools into one executable and distribute it to your team members, so they can run it 1:1 as you did.

As you probably already imagined, the only problem with this is a lot of typing. You got to provide so many arguments to mount all the locations and pass in all the environment variables. This is a lot of routine boring manual work.

This tool basically aims to remove that overhead. It will automatically discover all known file system locations and environment variables and pass them to the container so you don't have to. It is basically a smart wrapper for `docker run` or `kubernetes run` commands.

## Disclaimer

Be mindful of what you run.
This tool's mere purpose is to run the container as if it's internals were installed on your host.
That means the usual isolation attributed to the containers in general is compromised, any potential malware could easily escape to the host level and mess with your stuff. It will also have access to the docker on the host.
You should really know and trust containers you run with RunTainer as like you do it with regular software you install.

## How

This tool will do the following:

1. Discover and mount to your container well known locations (such as `~/.ssh`, `~/.aws` etc) as well as user-defined locations
1. Expose to your container well known environment variables (such as `AWS_PROFILE`, `AWS_DEFAULT_REGION` etc) as well as user-defined environment variables
1. Mount `cwd` and make it a `cwd` inside the container
1. Make UID/GID to match your host inside the container
2. Run the command you specified in your container (or default `ENTRYPOINT`)
