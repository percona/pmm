module github.com/percona/pmm

go 1.18

// Use for local development, but do not commit:
// replace github.com/percona/pmm => ../pmm

// Update depedencies with:
// go get -v github.com/percona/pmm@main

replace github.com/go-openapi/spec => github.com/Percona-Lab/spec v0.20.5-percona

replace gopkg.in/alecthomas/kingpin.v2 => github.com/Percona-Lab/kingpin v2.2.6-percona+incompatible

replace github.com/pganalyze/pg_query_go v1.0.3 => github.com/Percona-Lab/pg_query_go v1.0.3-percona

require (
	github.com/AlekSi/pointer v1.2.0
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d
	github.com/davecgh/go-spew v1.1.1
	github.com/go-openapi/errors v0.20.2
	github.com/go-openapi/runtime v0.24.1
	github.com/go-openapi/strfmt v0.21.2
	github.com/go-openapi/swag v0.21.1
	github.com/go-openapi/validate v0.21.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.10.0
	github.com/hashicorp/go-version v1.5.0
	github.com/lib/pq v1.10.6
	github.com/mwitkow/go-proto-validators v0.3.2
	github.com/percona/exporter_shared v0.7.3
	github.com/percona/go-mysql v0.0.0-20200630114833-b77f37c0bfa2
	github.com/percona/percona-toolkit v3.2.1+incompatible
	github.com/pganalyze/pg_query_go v1.0.3
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.2
	github.com/prometheus/common v0.34.0
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/objx v0.4.0
	github.com/stretchr/testify v1.7.1
	go.mongodb.org/mongo-driver v1.9.1
	golang.org/x/sys v0.0.0-20220412211240-33da011f77ad
	google.golang.org/genproto v0.0.0-20220414192740-2d67ff6cf2b4
	google.golang.org/grpc v1.47.0
	google.golang.org/protobuf v1.28.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/ini.v1 v1.66.6
	gopkg.in/reform.v1 v1.5.1
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

require (
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/go-openapi/analysis v0.21.3 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/loads v0.21.1 // indirect
	github.com/go-openapi/spec v0.20.5 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/montanaflynn/stats v0.0.0-20171201202039-1bf9dbcd8cbe // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.0.2 // indirect
	github.com/xdg-go/stringprep v1.0.2 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	golang.org/x/crypto v0.0.0-20201216223049-8b5274cf687f // indirect
	golang.org/x/net v0.0.0-20220425223048-2871e0cb64e4 // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/text v0.3.7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
