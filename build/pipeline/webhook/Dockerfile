# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.26
ARG ALPINE_VERSION=3.20

FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH}

RUN go build -trimpath -ldflags='-s -w' -o /out/webhook .

FROM alpine:${ALPINE_VERSION}

# make for the dispatched targets, ca-certificates for the GitHub API,
# tini for clean signal handling so child make processes get SIGTERM.
RUN apk add --no-cache make ca-certificates tini \
    && addgroup -S webhook \
    && adduser -S -G webhook -h /var/lib/webhook webhook \
    && mkdir -p /var/lib/webhook \
    && chown -R webhook:webhook /var/lib/webhook

COPY --from=builder /out/webhook /usr/local/bin/webhook

USER webhook
WORKDIR /var/lib/webhook

ENV LISTEN_ADDR=:7799 WORK_DIR=/var/lib/webhook

EXPOSE 7799

ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/webhook"]
