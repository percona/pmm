# PMM 2.x APIs.

All generated files (Go code, Swagger spec, documentation) are already stored in this repository.

## Browsing documentation

1. Generate TLS certificate for `nginx` for local testing (once):
```
brew install mkcert
mkcert -install
make cert
```

2. Serve API documentation with `nginx`:
```
make serve
```


## Updating APIs

1. Edit `.proto` files. Do not edit Swagger, `.pb.go`, `.pb.gw.go`. You can use `make clean` to remove all generated files.

2. Install `prototool` and fill `vendor/` (once):
```
make init
```

3. Generate files:
```
make gen
```
