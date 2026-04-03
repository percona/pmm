ARG PMM_SERVER_IMAGE="perconalab/pmm-server:3-dev-latest"
FROM $PMM_SERVER_IMAGE

ARG PMM_SERVER_IMAGE
ARG GO_VERSION="1.25.x"

USER root

RUN echo "Building with: GO: $GO_VERSION, PMM: $PMM_SERVER_IMAGE"
ENV GOPATH=/root/go
ENV PATH="$PATH:$GOPATH/bin"

RUN mkdir -p $GOPATH/src/github.com/percona/pmm
WORKDIR $GOPATH/src/github.com/percona/pmm

COPY ./ ./

RUN --mount=type=cache,target=/var/cache/dnf \
    python ./.devcontainer/setup.py

RUN mv -f $GOPATH/src/github.com/percona/pmm/bin/* $GOPATH/bin/

USER pmm
