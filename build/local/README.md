# Local builds

This directory contains a set of scripts aimed at providing a simple way to build PMM locally.

## Background

Historically, PMM used to be built using Jenkins. This worked well for the team, but not for the community. The learning curve was, and still is, rather steep, and it is hard for folks, even internally, to contribute to.

Therefore, we decided to make it possible to build PMM locally. This is a work in progress, but we are definitely committed to bring the developer experience to an acceptable level.

The build process is mostly based on bash scripts, which control the build flow. This was an intentional decision early on, since every developer should have at least a basic command of bash. Apart from bash and a few other well-known utilitites like `curl` or `make`, it also uses Docker for environment isolation and caching.

The build process is designed to run on a Linux host. We believe it can run on other flavors of Linux, including MacOS, with little to no modification (TBC).


## Prerequisites

Below is a list of prerequisites that are required to build PMM locally.

- OS: Linux (tested on Oracle Linux 9.3, CentOS 7, Ubuntu 22.04.3 LTS)
- Docker: 25.0.2+ (tested on 25.0.2)
- Docker Compose Plugin: 2.24.7+ (tested on 2.24.7)
- make: 4.3+ (tested on 4.3)
- bash (GNU): 5.1+ (tested on 5.1)
- git: 2.34.1+ (tested on 2.34.1)
- curl: 7.81.0+ (tested on 7.81.0)
- yq: 4.42.0+ (tested on 4.42.1)
- jq: 1.6+ (tested on 1.6)


## How to build PMM

1. Install the prerequisites
2. Clone the PMM repository to the user's home directory, e.g.: `git clone https://github.com/percona/pmm /home/user/pmm`. 
3. Change to the `build/local` directory in the cloned repo.
4. Run `build.sh --help` to print the help message and check the usage.
5. Run `build.sh` with parameters of your choice to build PMM v3.

Usually, you will want to rebuild PMM whenever there are changes in at least one of its components. All components of PMM are gathered together in one repository - `github.com/percona-lab/pmm-submodules` (or `pmm-submodules`). Therefore, you can run `build.sh` as often as those changes need to be factored in to the next build.

Once the build is finished, you can proceed with launching a new instance of PMM Server, or installing a freshly built PMM Client, and testing the changes.


## The `rpmbuild` image and docker cache

We use a special docker image to build various PMM artifacts - `perconalab/rpmbuild:3`. It comes provisioned with tools which are required to build PMM artifacts, for example RPM packages. As a build tool, it offers a number of benefits, two most obvious of which are:

- it frees the user from installing dependencies on their host machine
- it leverages a very powerful docker caching system, which results in reduced build times

During the first run, `build.sh` will create a few directories on the host machine, which are necessary to make use of docker cache. Please be aware, that the docker container's user needs to be able to write to these directories. The docker container's user is `builder` with uid 1000 and gid 1000. You need to make sure that the directories we create on the host are owned by a user with the same uid and gid. If the build fails, this is the first thing to check.


## Avoiding unnecessary builds

Sometimes, the changes you make affect only PMM Client. Other times, they affect only PMM Server. Therefore, you may want to skip building parts of PMM. The `build.sh` script offers several parameters to help control what you want to build.

* --no-update: run the build tasks without pulling the changes from `pmm-submodules` repository
* --update-only: pull changes from the repo without building PMM
* --no-client: do not build the client, use the cached PMM Client artifacts
* --no-client-docker: skip building PMM Client docker container
* --no-server-rpm: skip building PMM Server RPM artifacts
* --log-file <path>: change the path of the build log file

It's important to note, however, that once all changes are made and tested, you most probably want to re-build both PMM Client and Server to test them together.


## Target environments

Currently, local builds target the following environments:
- PMM Client
  - tarball - virtually any amd64 Linux environment
  - RPM - RHEL9-compatible environments
  - docker image - docker and Kubernetes environments (amd64)
- PMM Server
  - docker image - docker and Kubernetes environments (amd64)

