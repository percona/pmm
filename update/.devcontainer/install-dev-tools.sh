#!/bin/bash

# See https://code.visualstudio.com/docs/remote/remote-overview
# and https://code.visualstudio.com/docs/remote/containers.

set -m
set -o errexit
set -o xtrace

# download (in the background) the same verison as used by PMM build process
curl -sS https://dl.google.com/go/go1.17.8.linux-amd64.tar.gz -o /tmp/golang.tar.gz &

# to install man pages
sed -i '/nodocs/d' /etc/yum.conf

# enable experimental repository with latest development packages
sed -i'' -e 's^/release/^/experimental/^' /etc/yum.repos.d/pmm2-server.repo
percona-release enable original testing

# reinstall with man pages
yum install -y yum rpm
yum reinstall -y yum rpm

yum install -y gcc git make pkgconfig glibc-static \
    ansible-lint ansible \
    mc tmux psmisc lsof which iproute \
    bash-completion bash-completion-extras \
    man man-pages

fg || true
tar -C /usr/local -xzf /tmp/golang.tar.gz
update-alternatives --install "/usr/bin/go" "go" "/usr/local/go/bin/go" 0
update-alternatives --set go /usr/local/go/bin/go
update-alternatives --install "/usr/bin/gofmt" "gofmt" "/usr/local/go/bin/gofmt" 0
update-alternatives --set gofmt /usr/local/go/bin/gofmt
mkdir /root/go/bin
go env

# use modules to install (in the background) tagged releases
cd $(mktemp -d)
go mod init tools
env GOPROXY=https://proxy.golang.org go get -v \
    github.com/go-delve/delve/cmd/dlv@latest \
    golang.org/x/tools/gopls@latest &

cd /root/go/src/github.com/percona/pmm-update
make init

fg || true
