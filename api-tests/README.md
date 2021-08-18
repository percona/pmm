# pmm-api-tests

[![Build Status](https://travis-ci.com/Percona-Lab/pmm-api-tests.svg?branch=master)](https://travis-ci.com/Percona-Lab/pmm-api-tests)

API tests for PMM 2.x

# Setup Instructions

Make sure you have Go 1.16.x installed on your systems, execute the following steps
to setup API-tests in your local systems.

1. Run PMM Server.
2. Navigate to the tests root folder: `cd ~/go/src/github.com/percona/pmm-managed/api-tests`

# Usage

Run the tests using the following command:

```
go test ./... -pmm.server-url **pmm-server-url** -v
```

where `pmm-server-url` should be pointing to pmm-server.

# Docker

Build Docker image using the following command:

```
docker build -t IMAGENAME .
```

Run Docker container using the following command:

```
docker run -e PMM_SERVER_URL=**pmm-server-url** IMAGENAME
```

where `PMM_SERVER_URL` should be pointing to pmm-server.

If pmm-server located locally:

- Use --network=host while running docker container or add both containers to the same docker network.
- Use the insecure url if you default to a self-generated certificate.

# Contributing

All tests should follow these rules:

- Tests can work in parallel and in real system, so take into account that there might be records in database.
- Always revert changes made by test.
