# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2022-10-12

### BREAKING CHANGES

- Will no longer set `fsGroup` to the current host GID by default.
- Will now set `runAsUser` to the current host UID and `runAsGroup` to the current host GID instead.
- Added new `--run-as-current-user` and `--run-as-current-group` options to disable this new behavior.

Should be fine for most cases, but note that depending on what is in the image - it might break stuff sometimes.
Some images might rely on the user home directory. This new behavior is changing the user `id`, so the home directory bundled with the image no longer accessible.
In other cases, some software might not like that the current `id` set to the user that doesn't exists in the system. I.e.:

```bash
runtainer -q alpine whoami
whoami: unknown uid 1000
```

### Also in this release

- Updated all dependencies

## [0.1.6] - 2022-04-30

### Changed

- Force secure `0600` on the mounted secrets, but since we are using `fsGroup` - kubernetes will force `0640` to it.
- Fix error adding multiple `items` to `--secret-volume` by allowing them to be added individually as `item`.
  This usually warrants major version bump since it is not backward compatible, but it is a hotfix to a feature that was just released and never worked, so it's fine.
- Implement `--disable-discovery` that allows to disable all or one-by-one elements of automatic discovery

## [0.1.5] - 2022-04-30

### Added

- Add options to `--secret-env` and `--secret-volume`
- With `--secret-env` you can now specify custom `prefix`
- With `--secret-volume` you can now specify custom `mountPath` and `items`

## [0.1.4] - 2022-04-30

### Added

- Forward `$SSH_AUTH_SOCK` in to the container
- Implement secrets injection `--secret-env` and `--secret-volume`

## [0.1.3] - 2021-10-06

- Implemented `--secret`/`-S`

## [0.1.2] - 2021-09-28

- Fix terminal TTY (regression in `v0.1.1`)

## [0.1.1] - 2021-09-27

- Fixed terminal resizing and wrapping

## [0.1.0] - 2021-09-26

### BREAKING CHANGES

- Docker CLI is no longer used
- RT now uses K8s Go Client to run container in a pod
- There are substantial breaking changes in RT CLI arguments to back up that change

## [0.0.2] - 2020-08-25

### Added

- Added `--dry-run` mode to print out what it normally would run otherwise
- Implemented env variables discovery mechanism similar to what's for volumes
- `env` was moved out of `host` in config files - now it's a root level map
- [Discovery/AWS]: Add `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`, `AWS_ROLE_SESSION_NAME`, `AWS_STS_REGIONAL_ENDPOINTS` and `AWS_SDK_LOAD_CONFIG`
- Helm discovery was not enabled
- Implemented Terraform discovery

## [0.0.1] - 2020-08-22

### Added

- Initial release
