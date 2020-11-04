# Contributing notes

pmm-managed is highly integrated with PMM Server environment. As a result, development and testing is done partially inside PMM Server container which we call "devcontainer".


# Devcontainer setup

Install Docker and Docker Compose.

Run `make` to see a list of targets that can be run on host:
```
$ make
Please use `make <target>` where <target> is one of:
  env-up                    Start devcontainer.
  env-down                  Stop devcontainer.
  env                       Run `make TARGET` in devcontainer (`make env TARGET=help`); TARGET defaults to bash.
  release                   Build pmm-managed release binaries.
  help                      Display this help message.
```

`make env-up` starts devcontainer with all tools and mounts source code from the host. You can change code with your editor/IDE of choice as usual, or run editor inside devcontainer (see below for some special support of VSCode). To run make targets inside devcontainer use `make env TARGET=target-name`. For example:
```
$ make env TARGET=help
docker exec -it --workdir=/root/go/src/github.com/percona/pmm-managed pmm-managed-server make help
Please use `make <target>` where <target> is one of:
  gen                       Generate files.
  install                   Install pmm-managed binary.
  install-race              Install pmm-managed binary with race detector.
  test                      Run tests.
...
```
Run run tests, use `make env TARGET=test`, etc.

Alternatively, it is possible to run `make env` to get inside devcontainer and run make targets as usual:
```
$ make env
docker exec -it --workdir=/root/go/src/github.com/percona/pmm-managed pmm-managed-server make _bash
/bin/bash
[root@pmm-managed-server pmm-managed]# make test
make[1]: Entering directory `/root/go/src/github.com/percona/pmm-managed'
go test -timeout=30s -p 1 ./...
...
```

`run` and `run-race` targets replace `/usr/sbin/pmm-managed` and restart pmm-managed with `supervisorctl`. As a result, it will use normal filesystem locations (`/etc/victoriametrics-promscrape.yml`, `/etc/supervisord.d`, etc.) and `pmm-managed` PostgreSQL database. Other locations (inside `testdata`) and `pmm-managed-dev` database are used for unit tests.


## VSCode

VSCode provides first-class support for devcontainers. See:

* https://code.visualstudio.com/docs/remote/remote-overview
* https://code.visualstudio.com/docs/remote/containers


# Internals

There are three makefiles: `Makefile` (host), `Makefile.devcontainer`, and `Makefile.include` (included in both). `Makefile.devcontainer` is mounted on top of `Makefile` inside devcontainer (see `docker-compose.yml`) to enable `make env TARGET=target-name` usage.

Devcontainer initialization code is located in `.devcontainer/setup.py`. It uses multiprocessing to run several commands in parallel to speed-up setup.


# How to make PR

Before making PR, please run these commands locally:
* `make env TARGET=check-all` to run all checkers and linters.
* `make env TARGET=test-race` to run tests.
