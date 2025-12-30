ARG GO_VERSION=latest
FROM golang:${GO_VERSION}

# Install dependencies needed for building PMM components
RUN apt-get update && \
    apt-get install -y \
        zip \
        libc-dev \
        libkrb5-dev \
        libssl-dev \
    && rm -rf /var/lib/apt/lists/*

# Set up Go environment
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /build
