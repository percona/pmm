ARG PMM_SERVER_IMAGE="perconalab/pmm-server:dev-latest"
FROM $PMM_SERVER_IMAGE

ARG PMM_SERVER_IMAGE
ARG GO_VERSION="1.19.x"

RUN echo "Building with: GO: ${GO_VERSION}, PMM: ${PMM_SERVER_IMAGE}"

ENV PATH="/root/go/bin:${PATH}"

RUN mkdir -p $GOPATH/src/github.com/percona/pmm
WORKDIR $GOPATH/src/github.com/percona/pmm

COPY ./ ./
# setup.py, uses a task from Makefile.devcontainer but expect it to be in the fault file Makefile
COPY ./Makefile.devcontainer ./Makefile

RUN python ./.devcontainer/setup.py
RUN mv -f $GOPATH/src/github.com/percona/pmm/bin/* /root/go/bin/
