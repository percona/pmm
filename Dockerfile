# This Dockerfile used only for the API tests.

FROM golang:1.16

RUN mkdir -p $GOPATH/src/github.com/percona/pmm-managed/api-tests

WORKDIR $GOPATH/src/github.com/percona/pmm-managed/api-tests/
COPY api-tests/ $GOPATH/src/github.com/percona/pmm-managed/api-tests/
COPY go.mod go.sum $GOPATH/src/github.com/percona/pmm-managed/

CMD make init run-race
