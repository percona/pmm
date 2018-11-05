# pmm-api

[![Build Status](https://travis-ci.org/percona/pmm.svg?branch=master)](https://travis-ci.org/percona/pmm)

PMM 2.x APIs.

## Local setup

Generate TLS certificate for `nginx` for local testing:
```
brew install mkcert
mkcert -install
make cert
```

Install `prototool` and fill `vendor/`:
```
make init
```

Generate files:
```
make gen
```

Serve API documentation with `nginx`:
```
make serve
```
