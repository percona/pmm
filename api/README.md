# PMM 3.x API.

All generated files (Go code, Swagger spec, documentation) are stored in this directory.

## Browsing documentation

Serve API documentation with `nginx`:
```
make serve
```


## Updating API

1. Edit `.proto` files. Do not edit Swagger, `.pb.go`, `.pb.gw.go`. You can use `make clean` to remove all generated files.

2. Install required tools (once):
```
make init
```

3. Generate files:
```
make gen
```
