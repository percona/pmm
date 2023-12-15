FROM centos:7

# enable nodesource repo for nodejs
RUN curl -sL https://rpm.nodesource.com/setup_18.x | bash -
RUN curl -sL https://dl.yarnpkg.com/rpm/yarn.repo | tee /etc/yum.repos.d/yarn.repo

RUN yum update -y && \
    yum install -y --setopt=skip_missing_names_on_install=False \
    gcc gcc-c++ \
    nodejs \
    libtool libtool-ltdl \
    make cmake \
    git \
    pkgconfig \
    sudo \
    automake autoconf \
    rpmdevtools createrepo_c epel-release \
    yum-utils rpm-build \
    wget \
    glibc-static \
    yarn && \
    yum -y remove nodesource-release-el7-1.noarch && \
    yum clean all && rm -rf /var/cache/yum

# keep that format for easier search
ENV GO_VERSION 1.21.3
ENV GO_RELEASER_VERSION 1.21.2

RUN if [ `uname -i` == "x86_64" ]; then ARCH=amd64; else ARCH=arm64; fi && \
    wget --progress=dot:giga https://dl.google.com/go/go${GO_VERSION}.linux-${ARCH}.tar.gz -O /tmp/golang.tar.gz && \
    wget --progress=dot:giga https://github.com/goreleaser/goreleaser/releases/download/v${GO_RELEASER_VERSION}/goreleaser-${GO_RELEASER_VERSION}-1.`uname -i`.rpm -O /tmp/goreleaser.rpm && \
    tar -C /usr/local -xzf /tmp/golang.tar.gz && \
    yum install -y /tmp/goreleaser.rpm && \
    rm /tmp/golang.tar.gz /tmp/goreleaser.rpm

RUN update-alternatives --install "/usr/bin/go" "go" "/usr/local/go/bin/go" 0
RUN update-alternatives --set go /usr/local/go/bin/go
RUN update-alternatives --install "/usr/bin/gofmt" "gofmt" "/usr/local/go/bin/gofmt" 0
RUN update-alternatives --set gofmt /usr/local/go/bin/gofmt

RUN useradd builder -u 1000 -m -G users,wheel && \
    echo "builder ALL=(ALL:ALL) NOPASSWD:ALL" >> /etc/sudoers && \
    echo "# macros"                      >  /home/builder/.rpmmacros && \
    echo "%_topdir    /home/builder/rpm" >> /home/builder/.rpmmacros && \
    mkdir /home/builder/rpm && \
    chmod 755 /home/builder && \
    chown -R builder:builder /home/builder

USER builder

ENV FLAVOR=rpmbuild OS=centos DIST=el7
WORKDIR /home/builder/rpm
