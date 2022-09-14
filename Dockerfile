# This Dockerfile used only for the API tests.

FROM golang:1.19

RUN mkdir -p $GOPATH/src/github.com/percona/pmm

COPY . $GOPATH/src/github.com/percona/pmm/
WORKDIR $GOPATH/src/github.com/percona/pmm/api-tests/

CMD make init run-race
