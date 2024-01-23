#!/bin/bash

# See https://code.visualstudio.com/docs/remote/remote-overview
# and https://code.visualstudio.com/docs/remote/containers.

set -m
set -o errexit
set -o xtrace

# download (in the background) the same verison as used by PMM build process
curl -sS https://dl.google.com/go/go1.21.5.linux-amd64.tar.gz -o /tmp/golang.tar.gz &

# to install man pages
sed -i '/nodocs/d' /etc/yum.conf

# enable experimental repository with latest development packages
sed -i'' -e 's^/release/^/experimental/^' /etc/yum.repos.d/pmm-server.repo
percona-release enable original testing

RHEL=$(rpm --eval '%{rhel}')
if [ "$RHEL" = "7" ]; then
    # this mirror always fails, on both AWS and github
    echo "exclude=mirror.es.its.nyu.edu" >> /etc/yum/pluginconf.d/fastestmirror.conf
    yum clean plugins
    # https://stackoverflow.com/questions/26734777/yum-error-cannot-retrieve-metalink-for-repository-epel-please-verify-its-path
    sed -i "s/metalink=https/metalink=http/" /etc/yum.repos.d/epel.repo
fi

# reinstall with man pages
yum install -y yum rpm
yum reinstall -y yum rpm

yum install -y gcc git make pkgconfig \
    ansible \
    mc tmux psmisc lsof which iproute \
    bash-completion \
    man man-pages

if [ "$RHEL" = "7" ]; then
    yum install -y ansible-lint glibc-static bash-completion-extras
else
    yum install -y ansible-lint glibc-static --enablerepo=ol9_codeready_builder
fi

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
