# Contributing notes

## Local setup

You have to use Docker Compose to run most of the tests.

```sh
docker-compose up
make
```

Run it with

```sh
make run
```

Swagger UI will be available on http://127.0.0.1:7772/swagger/.

## Vendoring

We use [dep](https://github.com/golang/dep) to vendor dependencies.
