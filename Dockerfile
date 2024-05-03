# This Dockerfile is used only for API tests.

FROM golang:1.22

RUN export GOPATH=$(go env GOPATH) && \
    mkdir -p $GOPATH/src/github.com/percona/pmm

COPY . $GOPATH/src/github.com/percona/pmm/
WORKDIR $GOPATH/src/github.com/percona/pmm/api-tests/

CMD make init run-race
