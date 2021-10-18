module github.com/percona/pmm-managed

go 1.16

// Use for local development, but do not commit:
// replace github.com/percona/pmm => ../pmm
// replace github.com/percona-platform/saas => ../saas
// replace github.com/percona-platform/dbaas-api => ../dbaas-api

// Update depedencies with:
// go get -v github.com/percona/pmm@main
// go get -v github.com/percona-platform/saas@latest
// go get -v github.com/percona-platform/dbaas-api@latest

require (
	github.com/AlekSi/pointer v1.1.0
	github.com/Azure/azure-sdk-for-go v49.2.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.18
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.7
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/aws/aws-sdk-go v1.34.28
	github.com/brianvoe/gofakeit v3.18.0+incompatible
	github.com/brianvoe/gofakeit/v6 v6.4.0
	github.com/davecgh/go-spew v1.1.1
	github.com/go-co-op/gocron v1.7.1
	github.com/go-openapi/runtime v0.19.20
	github.com/go-openapi/spec v0.19.9 // indirect
	github.com/go-openapi/strfmt v0.19.5
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/protobuf v1.5.0
	github.com/google/uuid v1.1.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.15.0
	github.com/hashicorp/go-version v1.3.0
	github.com/lib/pq v1.8.0
	github.com/minio/minio-go/v7 v7.0.10
	github.com/percona-platform/dbaas-api v0.0.0-20211006103141-a68316e565d2
	github.com/percona-platform/saas v0.0.0-20210505183502-c18b6f47c932
	github.com/percona/pmm v0.0.0-20211006103121-6dee93c2029b
	github.com/percona/promconfig v0.2.1
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_golang v1.9.0
	github.com/prometheus/client_model v0.2.1-0.20200623203004-60555c9708c7 // indirect
	github.com/prometheus/common v0.15.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.7.0
	go.starlark.net v0.0.0-20201210151846-e81fc95f7bd5
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83 // indirect
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.26.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/reform.v1 v1.5.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)
