# This Dockerfile used only for the API tests.

FROM golang:1.18

RUN mkdir -p $GOPATH/src/github.com/percona/pmm/api-tests

WORKDIR $GOPATH/src/github.com/percona/pmm/api-tests/
COPY api-tests/ $GOPATH/src/github.com/percona/pmm/api-tests/
COPY go.mod go.sum $GOPATH/src/github.com/percona/pmm/

CMD make init run-race
