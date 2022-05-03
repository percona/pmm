//go:build tools

package tools

import (
	_ "github.com/BurntSushi/go-sumtype"
	_ "github.com/Percona-Lab/swagger-order"
	_ "github.com/bufbuild/buf/cmd/buf"
	_ "github.com/daixiang0/gci"
	_ "github.com/go-openapi/runtime/client"
	_ "github.com/go-openapi/spec"
	_ "github.com/go-swagger/go-swagger/cmd/swagger"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"
	_ "github.com/mwitkow/go-proto-validators/protoc-gen-govalidators"
	_ "github.com/quasilyte/go-consistent"
	_ "golang.org/x/tools/cmd/goimports"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
	_ "mvdan.cc/gofumpt"
)

//go:generate go build -o ../bin/buf github.com/bufbuild/buf/cmd/buf
//go:generate go build -o ../bin/gci github.com/daixiang0/gci
//go:generate go build -o ../bin/go-consistent github.com/quasilyte/go-consistent
//go:generate go build -o ../bin/go-sumtype github.com/BurntSushi/go-sumtype
//go:generate go build -o ../bin/gofumpt mvdan.cc/gofumpt
//go:generate go build -o ../bin/goimports golang.org/x/tools/cmd/goimports
//go:generate go build -o ../bin/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint
//go:generate go build -o ../bin/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
//go:generate go build -o ../bin/protoc-gen-go-grpc google.golang.org/grpc/cmd/protoc-gen-go-grpc
//go:generate go build -o ../bin/protoc-gen-govalidators github.com/mwitkow/go-proto-validators/protoc-gen-govalidators
//go:generate go build -o ../bin/protoc-gen-grpc-gateway github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway
//go:generate go build -o ../bin/protoc-gen-openapiv2 github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2
//go:generate go build -o ../bin/swagger github.com/go-swagger/go-swagger/cmd/swagger
//go:generate go build -o ../bin/swagger-order github.com/Percona-Lab/swagger-order
