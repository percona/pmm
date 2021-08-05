module github.com/percona/qan-api2

go 1.15

require (
	github.com/ClickHouse/clickhouse-go v1.4.5
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/golang-migrate/migrate v3.5.4+incompatible
	github.com/golang/protobuf v1.4.3
	github.com/google/uuid v1.1.1
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.15.0
	github.com/jmoiron/sqlx v1.2.0
	github.com/percona/pmm v0.0.0-20210629094649-72e6ecae869b
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.1.0
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	golang.org/x/sys v0.0.0-20200724161237-0e2f3a69832c
	google.golang.org/grpc v1.32.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
)
