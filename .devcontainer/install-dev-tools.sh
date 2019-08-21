#!/bin/bash

# See https://code.visualstudio.com/docs/remote/remote-overview
# and https://code.visualstudio.com/docs/remote/containers.

set -o errexit
set -o xtrace

# download (in the background) the same verison as used by PMM build process
curl -sS https://dl.google.com/go/go1.12.9.linux-amd64.tar.gz -o /tmp/golang.tar.gz &

# to install man pages
sed -i '/nodocs/d' /etc/yum.conf

# reinstall with man pages
yum reinstall -y yum rpm

yum install -y gcc git make pkgconfig glibc-static \
    ansible-lint \
    mc tmux psmisc which iproute \
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

curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

# use modules to install (in the background) tagged releases
cd $(mktemp -d)
go mod init tools
go get -v golang.org/x/tools/cmd/gopls \
    github.com/acroca/go-symbols \
    github.com/go-delve/delve/cmd/dlv \
    github.com/ramya-rao-a/go-outline &

cd /root/go/src/github.com/percona/pmm-update
make init

fg || true
