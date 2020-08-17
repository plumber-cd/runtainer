# runtainer

Run anything as a container

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
This tool's mere purpose is to run the container as if it's internals were run on your host.
That means the usual isolation attributed to the containers in general is compromised, any potential malware could easily escape to the host level and mess with your stuff.
You should really know and trust containers you run.

## How

This tool will do the following:

1. Discover and mount to your container well known locations (such as `~/.ssh`, `~/.aws` etc) as well as user-defined locations
1. Expose to your container well known environment variables (such as `AWS_PROFILE`, `AWS_DEFAULT_REGION` etc) as well as user-defined environment variables
1. Mount `cwd` and make it a `cwd` inside the container
1. Make UID/GID to match your host inside the container
1. Run the command you specified in your container (or default `ENTRYPOINT`)

## Examples

```bash
runtainer maven:3.6.3-jdk-14 -- mvn version
runtainer -d /tmp maven:3.6.3-jdk-14 -- mvn version
runtainer maven:3.6.3-jdk-14 -e PROFILE -- mvn version
```
