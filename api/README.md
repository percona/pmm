# PMM 2.x APIs.

All generated files (Go code, Swagger spec, documentation) are already stored in this repository.

## Browsing documentation

Serve API documentation with `nginx`:
```
make serve
```


## Updating APIs

1. Edit `.proto` files. Do not edit Swagger, `.pb.go`, `.pb.gw.go`. You can use `make clean` to remove all generated files.

2. Install `prototool` and other required tools (once):
```
make init
```

3. Generate files:
```
make gen
```


## Alertmanager

`alertmanager/openapi.yaml` is copied from https://github.com/prometheus/alertmanager/blob/master/api/v2/openapi.yaml.
Then Swagger client is generated using `make gen-alertmanager`.
