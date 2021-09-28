# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
