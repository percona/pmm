# PMM 2.x APIs.

All generated files (Go code, Swagger spec, documentation) are already stored in this repository.

## Browsing documentation

Serve API documentation with `nginx`:
```
make serve
```


## Updating APIs

1. Edit `.proto` files. Do not edit Swagger, `.pb.go`, `.pb.gw.go`. You can use `make clean` to remove all generated files.

2. Install required tools (once):
```
make init
```

3. Generate files:
```
make gen
```
