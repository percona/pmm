# Local builds

This directory contains a set of scripts aimed at providing a simple way to build PMM locally.

## Background

Historically, PMM used to be built using Jenkins. This worked well for the team, but not for the community. The learning curve was, and still is, rather steep, and it is hard for folks, even internally, to contribute to.

Therefore, we decided to make it possible to build PMM locally. This is a work in progress, but we are definitely committed to bring the developer experience to an acceptable level.

The build process is mostly based on bash scripts, which control the build flow. This was an intentional decision early on, since every developer should have at least a basic command of bash. Apart from bash and a few other well-known utilitites like `curl` or `make`, it also uses Docker for environment isolation and caching.

The build process is designed to run on Linux or MacOS. We believe it could be adapated to run on other types of operating systems with little to no modification.


## Prerequisites

Below is a list of prerequisites that are required to build PMM locally.

- OS: Linux (tested on Oracle Linux 9.3, Ubuntu 22.04.3 LTS), MacOS (tested on Sequoia 15.1)
- Docker 25.0.2+
- Docker [buildx plugin](https://github.com/docker/buildx) 0.16.0+
- make
- bash
- tar
- git
- curl

Also, it is required to define an environment variable `GITHUB_API_TOKEN` with a valid GitHub Personal Access Token (PAT). This token is used to pull the changes from the `percona-lab/pmm-submodules` and other project repositories. The token must have the following permissions:

- repo:status
- public_repo
- read:user

Please note, that building some of the PMM internals, such as Grafana, requires at least 8GB of memory available to docker. The number of CPUs, however, does not matter that much.

## How to use this script to build PMM

1. Install the prerequisites
2. Clone the `pmm` repository - `git clone git@github.com:percona/pmm`.
3. Change directory to `pmm` - `cd pmm`.
4. Run `./build --help` to display the script usage.
5. Run `./build --init` to provision all dependent submodules using `https://github.com/percona-lab/pmm-submodules` repository.
6. Run `./build` with parameters of your choice to build PMM v3.

Usually, you will want to rebuild PMM whenever there are changes in at least one of its components. With the exception of ClickHouse and All components of PMM are gathered together in one repository - `github.com/percona-lab/pmm-submodules` (or `pmm-submodules`). Therefore, you can run `build.sh` as often as those changes need to be factored in to the next build.

Once the build is finished, you can proceed with launching a new instance of PMM Server, or installing a freshly built PMM Client, and testing the changes.


## The `rpmbuild` image and docker cache

We use a special docker image to build various PMM artifacts - `perconalab/rpmbuild:3`. It comes provisioned with tools which are required to build PMM artifacts, for example RPM packages. As a build tool, it offers a number of benefits, two most obvious of which are:

- it frees the user from installing dependencies on their machine
- it leverages the docker caching, which results in much reduced build times

During the first run, `build.sh` will create a few directories on the host machine, which are necessary to make use of docker cache. Please be aware, that the docker container's user needs to be able to write to these directories. The docker container's user is `builder` with uid 1000 and gid 1000. You need to make sure that the directories the script creates on the host are owned by a user with the same uid and gid. If the build fails, this is the first thing to check.

## Using cache to speed up builds

In order to save time and to avoid building the same package versions repeatedly when there are no code changes, we use a combination of file based and docker based cache. The idea is simple:

- before proceeding with building a package, we check if this package version can be found locally; if that's the case, we reuse the package instead of building it;
- if the package can not be found, we build it and store in cache.

There is also a special environment variable `CI`, which controls whether the cache should be stored in Percona's AWS S3 bucket. The use of this variable is reserved for Percona's internal use. Please note, that using S3 cache to build PMM locally is not currently supported.

## Avoiding unnecessary builds

Sometimes, the changes you make affect only PMM Client. Other times, they affect only PMM Server. Therefore, you may want to skip building parts of PMM. The `build` script offers several parameters to help control what you want to build.

* --no-update: run the build tasks without pulling the changes from `pmm-submodules` repository
* --update-only: pull changes from the repo without building PMM
* --client-only: build PMM Client only, skip building the Server
* --no-client: do not build the client, use the cached PMM Client artifacts
* --no-client-docker: skip building PMM Client docker container
* --log-file <path>: change the path of the build log file

It's important to note, however, that once all changes are made and tested, you most probably want to re-build both PMM Client and Server to test them together.


## Target environments

Currently, Local Builds target the following platforms and distributions:

### PMM Client

| Platform     | tarball | rpm, rpms | deb, deb-src | docker image |
|--------------|:-------:|:---------:|:------------:|:------------:|
| linux/amd64  |    X    |     X     |      X       |      X       |
| linux/arm64  |    X    |     X     |      X       |      X       |
| darwin/arm64 |    X    |     -     |      -       |      -       |

### PMM Server

| Platform         | docker image |
|------------------|:------------:|
| linux/amd64      |      X       |
| linux/arm64      |      X       |
| darwin/arm64     |      -       |



## TODO

* download the sources to a local directory `.modules` w/o using pmm-submodules ✅
* have a `sbom.json` file containing the bill of all repositories, such as grafana, exporters, etc. containing the following attributes:
  * component name
  * the repository URL
  * the branch used for the build
  * the path to the repository on disk
  * the commit hash
* provide better caching for components, which reside in one monorepo, by calculating a sha256sum on their directories:
  - pmm-ui ✅
  - pmm-qan
  - pmm-agent
  - pmm-admin
  - pmm-managed
  - pmm-vmproxy
* use the `--debug` parameter to control the verbosity of the logs (1/2 ✅)
* remove `jq` from prerequisites ✅
* do not require `ci.yml` to be present, generate it based on the current branch name of this (percona/pmm) repository ✅
* output the build summary at the end of the build
* implement the `--release` parameter
* implement the `--clean` parameter
* move the builds and the cache from the host to the container, fully isolating the build process
