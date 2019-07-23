#!/bin/bash

set -o errexit
set -o xtrace

yum install -y golang mc tmux which

go env

go get golang.org/x/tools/cmd/gopls \
    github.com/acroca/go-symbols \
    github.com/ramya-rao-a/go-outline

curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
