# Local builds

This directory contains a set of scripts aimed at providing a simple way to build PMM locally.

## Background

Historically, PMM used to be built using Jenkins. This worked well for the team, but not for the community. The learning curve was, and still is, rather steep, and it is hard for folks, even internally, to contribute to.

Therefore, we decided to make it possible to build PMM locally. This is a work in progress, but we are definitely committed to bring the developer experience to an acceptable level.

The build process is mostly based on bash scripts, which control the build flow. This was an intentional decision early on, since every developer should have at least a basic command of bash. Apart from bash and a few other well-known utilitites like `curl` or `make`, it also uses Docker for environment isolation and caching.

The build process is designed to run on Linux or MacOS. We believe it could be adapated to run on other types of operating systems with little to no modification.


## Prerequisites

Below is a list of prerequisites that are required to build PMM locally.

- OS: Linux (tested on Oracle Linux 9.3, CentOS 7, Ubuntu 22.04.3 LTS), MacOS (tested on Sequoia 15.1)
- Docker: 25.0.2+
- Docker buildx plugin: 0.16.0+, https://github.com/docker/buildx
- make
- bash (GNU)
- tar
- git
- curl
- jq: 1.6+


## How to use this script to build PMM

1. Install the prerequisites
2. Download the `build` script to the user's home directory, e.g.: `curl -o $HOME/build https://raw.githubusercontent.com/percona/pmm/refs/heads/PMM-13487-build-pmm-locally/build/local/build`. 
3. Change to the user `cd $HOME` directory.
4. Run `./build --provision` to download the source code using our `https://github.com/percona-lab/pmm-submodules` repository.
5. Run `./build --help` to display the script usage.
6. Run `./build` with parameters of your choice to build PMM v3.

Usually, you will want to rebuild PMM whenever there are changes in at least one of its components. All components of PMM are gathered together in one repository - `github.com/percona-lab/pmm-submodules` (or `pmm-submodules`). Therefore, you can run `build` as often as those changes need to be factored in to the next build.

Once the build is finished, you can proceed with launching a new instance of PMM Server, or installing a freshly built PMM Client, and testing the changes.


## The `rpmbuild` image and docker cache

We use a special docker image to build various PMM artifacts - `perconalab/rpmbuild:3`. It comes provisioned with tools which are required to build PMM artifacts, for example RPM packages. As a build tool, it offers a number of benefits, two most obvious of which are:

- it frees the user from installing dependencies on their host machine
- it leverages a very powerful docker caching system, which results in reduced build times

During the first run, `build` will create a few directories on the host machine, which are necessary to make use of docker cache. Please be aware, that the docker container's user needs to be able to write to these directories. The docker container's user is `builder` with uid 1000 and gid 1000. You need to make sure that the directories we create on the host are owned by a user with the same uid and gid. If the build fails, this is the first thing to check.

## Using S3 to cache packages

In order to save time and to avoid building the same package versions repeatedly, we use a dedicated AWS S3 bucket for caching in the following manner:

- before proceeding to building a package, we check if this package version can be found in S3 and we download the package instead of building it;
- if the package can not be found, we build and upload it to S3 for future reuse.

There is special variable `LOCAL_BUILD`, which needs to be set to '1' in case you don't have AWS CLI installed or you don't want to use the cache. Please be aware, that interacting with Percona's AWS S3 account, i.e. upload and download artefacts, requires authentication and is therefore reserved for Percona's own purposes. This is why, when building packages locally, you are requested to set this variable to '1', which happens to be the default value. Please note, that an attempt to interact with the S3 bucket without proper authorization will lead to a build failure.

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

Currently, local builds target the following environments:
- PMM Client
  - tarball - most Linux environments, noarch
  - RPM - RHEL9-compatible environments, amd64/aarch64
  - docker image - docker and Kubernetes environments amd64/aarch64
- PMM Server
  - docker image - docker and Kubernetes environments amd64/aarch64


## Ideas to evaluate

* download the sources to a local directory `.modules` w/o using pmm-submodules
* have a VERSION file, similar to the one in https://github.com/percona-lab/pmm-submodules/blob/v3/VERSION
* have a package.yml file containing hashes of all build artefacts (PMM components, i.e. grafana, pmm-managed, etc.)
* copy all build scripts to `build/local` to isolate the new build flows from the old ones
* use the `--debug` parameter to control the verbosity of the logs
* provide better caching for pmm-qan, pmm-agent, vmproxy and pmm-admin, which all reside in one monorepo, by calculating a sha256sum on their directories