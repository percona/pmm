all: test

PACKAGES := $(shell go list ./... | grep -v vendor)

# installs tools to $GOPATH/bin which is expected to be in $PATH
init:
	go install -v ./vendor/github.com/prometheus/prometheus/cmd/promtool
	go install -v ./vendor/github.com/golang/protobuf/protoc-gen-go
	go install -v ./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	go install -v ./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger

install:
	go install -v $(PACKAGES)
	go test -v -i $(PACKAGES)

test: install
	go test -v $(PACKAGES)

protos:  # make protos, not protoss
	rm -f api/*.pb.* api/*.json
	protoc -I api/ -Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		api/*.proto --go_out=plugins=grpc:api
	protoc -I api/ -Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		api/*.proto --grpc-gateway_out=logtostderr=true,request_context=true:api
	protoc -I api/ -Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		api/*.proto --swagger_out=logtostderr=true:api
	go install -v github.com/Percona-Lab/pmm-managed/api
