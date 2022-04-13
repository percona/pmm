module github.com/percona/pmm

go 1.16

replace github.com/go-openapi/spec => github.com/Percona-Lab/spec v0.19.8-percona

require (
	github.com/go-openapi/errors v0.19.6
	github.com/go-openapi/jsonreference v0.19.4 // indirect
	github.com/go-openapi/runtime v0.19.20
	github.com/go-openapi/strfmt v0.19.5
	github.com/go-openapi/swag v0.19.9
	github.com/go-openapi/validate v0.19.10
	github.com/golang/protobuf v1.5.2
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.10.0
	github.com/mitchellh/mapstructure v1.3.3 // indirect
	github.com/mwitkow/go-proto-validators v0.3.2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.1
	github.com/stretchr/testify v1.7.1
	go.mongodb.org/mongo-driver v1.7.1
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9
	google.golang.org/genproto v0.0.0-20220317150908-0efb43f6373e
	google.golang.org/grpc v1.45.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/reform.v1 v1.5.1
)
