FROM golang:1.16

RUN mkdir -p $GOPATH/src/github.com/percona/pmm-managed/api-tests

WORKDIR $GOPATH/src/github.com/percona/pmm-managed/api-tests/
COPY . $GOPATH/src/github.com/percona/pmm-managed/api-tests/

CMD make init run-race
