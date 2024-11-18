#!/bin/bash

# See https://code.visualstudio.com/docs/remote/remote-overview
# and https://code.visualstudio.com/docs/remote/containers.

set -m
set -o errexit
set -o xtrace

# download (in the background) the same verison as used by PMM build process
curl -sSL https://dl.google.com/go/go1.23.2.linux-amd64.tar.gz -o /tmp/golang.tar.gz &

# to install man pages
sed -i '/nodocs/d' /etc/yum.conf

yum update -y percona-release
percona-release enable pmm3-client testing

# reinstall with man pages
yum install -y yum rpm
yum reinstall -y yum rpm

yum install -y git make pkgconfig ansible \
    mc tmux psmisc lsof which iproute \
    bash-completion \
    man man-pages

yum --enablerepo=ol9_codeready_builder install -y gcc glibc-static ansible-lint

fg || true
tar -C /usr/local -xzf /tmp/golang.tar.gz && rm -f /tmp/golang.tar.gz
update-alternatives --install "/usr/bin/go" "go" "/usr/local/go/bin/go" 0
update-alternatives --set go /usr/local/go/bin/go
update-alternatives --install "/usr/bin/gofmt" "gofmt" "/usr/local/go/bin/gofmt" 0
update-alternatives --set gofmt /usr/local/go/bin/gofmt
mkdir /root/go/bin
go env

# use modules to install (in the background) tagged releases
cd $(mktemp -d)
go mod init tools
env GOPROXY=https://proxy.golang.org go get -v github.com/go-delve/delve/cmd/dlv@latest &

cd /root/go/src/github.com/percona/pmm
make init

fg || true
