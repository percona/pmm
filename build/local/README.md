# Local builds

This directory contains a set of scripts aimed at providing a simple way to build PMM locally.

## Background

Historically, PMM used to be built using Jenkins. This worked well for the team, but not for the community. The learning curve was, and still is, rather steep, and it is hard for folks, even internally, to contribute to.

Therefore, we decided to make it possible to build PMM locally. This is a work in progress, but we are definitely committed to making it easier to build PMM locally.

The build process is mostly based on bash scripts, which control the build flow. This was an intentional decision early on to make the build process easy to understand and contribute to. Apart from bash and a few other well-known utilitites like `curl` or `make`, it also uses Docker for environment isolation and caching.

The build process is designed to be run on a Linux host. We believe it can be run on other flavors of Linux, including MacOS, with little to no modification (TBC).

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

## Build Steps

1. Install the prerequisites
2. Clone the PMM repository
3. Change to the `build/local` directory
4. Run the `build.sh` script
