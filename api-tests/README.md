# pmm-api-tests

API tests for PMM 3.x

# Setup Instructions

Make sure you have the latest Go version installed on your systems, execute the following steps
to set up API-tests in your local systems.

1. Run PMM Server. This can be done by running `make env-up` in the root (`pmm`) directory.
2. Replace `$PMM_SERVER_URL` with a URL in format `https://USERNAME:PASSWORD@HOST`. For local development it's usually `https://admin:admin@127.0.0.1`.

# Usage

Precompile tests using the following command:
```
make init
```

Run the tests using the following command:

```
PMM_SERVER_URL=$PMM_SERVER_URL make test
```

# Docker

Build Docker image using the following command:

```
make docker-build-image
```

Run Docker container using the following command:

```
PMM_SERVER_URL=$PMM_SERVER_URL make docker-run-tests
```

where `PMM_SERVER_URL` should be pointing to a running PMM Server.

If pmm-server is located locally:

- Use --network=host while running docker container or add both containers to the same docker network.
- Use the insecure url if you default to a self-generated certificate.

# Contributing

All tests should follow these rules:

- Tests can work in parallel and on a real system, so take into account that there might be records in database.
- Always revert changes made by tests.
