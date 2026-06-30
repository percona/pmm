# This Dockerfile is used only for API tests.
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown
ARG GO_VERSION

FROM golang:$GO_VERSION AS builder
ENV GONOPROXY='github.com/percona,github.com/Percona-Lab'
ENV GONOSUMDB='github.com/percona,github.com/Percona-Lab'
ENV GOPRIVATE='github.com/percona,github.com/Percona-Lab'

WORKDIR $GOPATH/pmm

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . $GOPATH/pmm
WORKDIR $GOPATH/pmm/api-tests/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    make -j4 init

CMD ["make", "test-report"]
