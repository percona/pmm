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

# enable experimental repository with latest development packages
sed -i'' -e 's^/release/^/experimental/^' /etc/yum.repos.d/pmm2-server.repo

RHEL=$(rpm --eval '%{rhel}')
if [ "$RHEL" = "7" ]; then
    # disable fastestmirror plugin, which mostly fails due to CentOS 7 being EOL
    sed -i 's/enabled=1/enabled=0/g' /etc/yum/pluginconf.d/fastestmirror.conf

    if [ "$PMM_SERVER_IMAGE" = "percona/pmm-server:2.0.0" ]; then
      sed -i -e 's/^\(mirrorlist\)/#\1/g' /etc/yum.repos.d/CentOS-Base.repo
      sed -i -e 's/^#baseurl.*/baseurl=http:\/\/vault.centos.org\/centos\/$releasever\/os\/$basearch\//g' /etc/yum.repos.d/CentOS-Base.repo

      # https://stackoverflow.com/questions/26734777/yum-error-cannot-retrieve-metalink-for-repository-epel-please-verify-its-path
      yum --disablerepo=epel install -y ca-certificates-2020.2.41
      yum install -y gcc-4.8.5 glibc-static-2.17.317
    else
      yum --disablerepo=epel update -y ca-certificates
      yum --disablerepo="*" --enablerepo=base --enablerepo=updates install -y gcc glibc-static

      sed -i -e 's/^\(mirrorlist\)/#\1/g' /etc/yum.repos.d/CentOS-Base.repo
      sed -i -e 's/^#baseurl.*/baseurl=http:\/\/vault.centos.org\/centos\/$releasever\/os\/$basearch\//g' /etc/yum.repos.d/CentOS-Base.repo
    fi

    yum clean all
    yum makecache fast
fi

yum update -y percona-release
percona-release enable pmm2-client testing

# reinstall with man pages
yum install -y yum rpm
yum reinstall -y yum rpm

yum install -y git make pkgconfig ansible \
    mc tmux psmisc lsof which iproute \
    bash-completion \
    man man-pages

if [ "$RHEL" = '7' ]; then
    yum install -y ansible-lint bash-completion-extras
else
    yum --enablerepo=ol9_codeready_builder install -y gcc glibc-static ansible-lint
fi

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
env GOPROXY=https://proxy.golang.org go get -v github.com/go-delve/delve/cmd/dlv@latest &

cd /root/go/src/github.com/percona/pmm
make init

fg || true
