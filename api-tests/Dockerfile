# This Dockerfile is used only for API tests.

FROM golang:1.25

RUN export GOPATH=$(go env GOPATH) && \
    mkdir -p $GOPATH/pmm

COPY . $GOPATH/pmm/
WORKDIR $GOPATH/pmm/api-tests/

CMD ["make", "init", "run-race"]
