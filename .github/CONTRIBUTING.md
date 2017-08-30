# Contributing notes

## Local setup

You have to use Docker Compose to run most of the tests.

```
docker-compose up
make
```

## Vendoring

We use [dep](https://github.com/golang/dep) to vendor dependencies.
