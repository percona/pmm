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
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/protobuf v1.4.3
	github.com/grpc-ecosystem/grpc-gateway v1.15.0
	github.com/mitchellh/mapstructure v1.3.3 // indirect
	github.com/mwitkow/go-proto-validators v0.3.2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/common v0.10.0 // indirect
	github.com/prometheus/procfs v0.1.3 // indirect
	github.com/stretchr/testify v1.6.1
	go.mongodb.org/mongo-driver v1.7.1
	golang.org/x/net v0.0.0-20200707034311-ab3426394381 // indirect
	golang.org/x/sys v0.0.0-20200722175500-76b94024e4b6
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
	google.golang.org/grpc v1.32.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/reform.v1 v1.5.0
)
