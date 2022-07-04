ARG PMM_VERSION="percona/pmm-server:2.28.0"
FROM $PMM_VERSION

ARG PMM_VERSION
ARG GO_VERSION="1.18.x"

RUN echo "Building with: GO: ${GO_VERSION}, PMM: ${PMM_VERSION}"

ENV PATH="/root/go/bin:${PATH}"

RUN mkdir -p $GOPATH/src/github.com/percona/pmm
WORKDIR $GOPATH/src/github.com/percona/pmm

COPY ./ ./
# setup.py, uses a task from Makefile.devcontainer but expect it to be in the fault file Makefile
COPY ./Makefile.devcontainer ./Makefile

RUN python ./.devcontainer/setup-ex.py
RUN mv -f $GOPATH/src/github.com/percona/pmm/bin/* /root/go/bin/
