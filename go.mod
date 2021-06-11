module github.com/percona/pmm-agent

go 1.16

replace gopkg.in/alecthomas/kingpin.v2 => github.com/Percona-Lab/kingpin v2.2.6-percona+incompatible

replace github.com/lfittl/pg_query_go v1.0.2 => github.com/Percona-Lab/pg_query_go v1.0.1-0.20190723081422-3fc3af54a6f7

require (
	github.com/AlekSi/pointer v1.1.0
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/go-openapi/runtime v0.19.20
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/protobuf v1.4.3
	github.com/google/uuid v1.1.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.15.1
	github.com/hashicorp/go-version v1.3.0
	github.com/konsorten/go-windows-terminal-sequences v1.0.3 // indirect
	github.com/lfittl/pg_query_go v1.0.2
	github.com/lib/pq v1.10.0
	github.com/percona/exporter_shared v0.7.2
	github.com/percona/go-mysql v0.0.0-20200630114833-b77f37c0bfa2
	github.com/percona/percona-toolkit v3.2.1+incompatible
	github.com/percona/pmm v0.0.0-20210611190936-96a88244ad87
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.1
	github.com/prometheus/common v0.10.0
	github.com/prometheus/procfs v0.5.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/objx v0.2.0
	github.com/stretchr/testify v1.6.1
	go.mongodb.org/mongo-driver v1.3.5
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/sys v0.0.0-20200722175500-76b94024e4b6
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
	google.golang.org/grpc v1.32.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/reform.v1 v1.5.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)
