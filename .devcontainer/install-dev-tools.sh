#!/bin/bash

# See https://code.visualstudio.com/docs/remote/remote-overview
# and https://code.visualstudio.com/docs/remote/containers.

set -o errexit
set -o xtrace

yum install -y golang mc tmux which bash-completion bash-completion-extras

go env

go get golang.org/x/tools/cmd/gopls \
    github.com/acroca/go-symbols \
    github.com/go-delve/delve/cmd/dlv \
    github.com/ramya-rao-a/go-outline

curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

cd /root/go/src/github.com/percona/pmm-update
make init
