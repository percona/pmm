//go:build tools
// +build tools

package tools

import (
	_ "github.com/golang/protobuf/protoc-gen-go"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
	_ "github.com/mwitkow/go-proto-validators/protoc-gen-govalidators"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger"
	_ "github.com/go-openapi/runtime/client"
	_ "github.com/go-openapi/spec"
	_ "github.com/go-swagger/go-swagger/cmd/swagger"
	_ "github.com/BurntSushi/go-sumtype"
	_ "github.com/Percona-Lab/swagger-order"

)