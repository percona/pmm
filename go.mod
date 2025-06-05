module github.com/percona/pmm

go 1.24.3

// Update saas with
// go get -v github.com/percona/saas@latest

replace github.com/grpc-ecosystem/go-grpc-prometheus => github.com/Percona-Lab/go-grpc-prometheus v0.0.0-20230116133345-3487748d4592

replace github.com/go-openapi/spec => github.com/Percona-Lab/spec v0.21.0-percona

replace github.com/alecthomas/kingpin/v2 => github.com/Percona-Lab/kingpin/v2 v2.4.0-percona

replace golang.org/x/crypto => github.com/percona-lab/crypto v0.0.0-20231108144114-756dfb24eaf2

require (
	github.com/AlekSi/pointer v1.2.0
	github.com/ClickHouse/clickhouse-go/v2 v2.35.0
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/alecthomas/kingpin/v2 v2.4.0
	github.com/alecthomas/kong v1.11.0
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b
	github.com/aws/aws-sdk-go-v2 v1.36.3
	github.com/aws/aws-sdk-go-v2/config v1.29.14
	github.com/aws/aws-sdk-go-v2/credentials v1.17.67
	github.com/aws/aws-sdk-go-v2/service/rds v1.96.0
	github.com/aws/smithy-go v1.22.2
	github.com/blang/semver v3.5.1+incompatible
	github.com/brianvoe/gofakeit/v6 v6.28.0
	github.com/charmbracelet/bubbles v0.21.0
	github.com/charmbracelet/bubbletea v1.3.4
	github.com/charmbracelet/lipgloss v1.1.0
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/docker/docker v28.1.1+incompatible
	github.com/docker/go-connections v0.5.0
	github.com/envoyproxy/protoc-gen-validate v1.2.1
	github.com/go-co-op/gocron v1.37.0
	github.com/go-openapi/errors v0.22.0
	github.com/go-openapi/runtime v0.28.0
	github.com/go-openapi/strfmt v0.23.0
	github.com/go-openapi/swag v0.23.0
	github.com/go-openapi/validate v0.24.0
	github.com/go-sql-driver/mysql v1.9.0
	github.com/gogo/status v1.1.1
	github.com/golang-migrate/migrate/v4 v4.18.1
	github.com/google/tink/go v1.7.0
	github.com/google/uuid v1.6.0
	github.com/grafana/grafana-api-golang-client v0.27.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.3
	github.com/hashicorp/go-version v1.7.0
	github.com/hashicorp/raft v1.7.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/jotaen/kong-completion v0.0.5
	github.com/lib/pq v1.10.9
	github.com/minio/minio-go/v7 v7.0.55
	github.com/percona/exporter_shared v0.7.5
	github.com/percona/go-mysql v0.0.0-20250429063950-bc274ddba990
	github.com/percona/percona-toolkit v0.0.0-20250514130928-84095fd7d7d9
	github.com/percona/promconfig v0.2.5
	github.com/percona/saas v0.0.0-20240923141535-da19f6682c6e
	github.com/pganalyze/pg_query_go/v6 v6.1.0
	github.com/pkg/errors v0.9.1
	github.com/pkg/sftp v1.13.6
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2
	github.com/prometheus/client_golang v1.22.0
	github.com/prometheus/common v0.64.0
	github.com/prometheus/prometheus v0.304.0
	github.com/ramr/go-reaper v0.2.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/objx v0.5.2
	github.com/stretchr/testify v1.10.0
	github.com/tink-crypto/tink-go v0.0.0-20230613075026-d6de17e3f164
	go.mongodb.org/mongo-driver v1.17.3
	go.starlark.net v0.0.0-20230717150657-8a3343210976
	golang.org/x/crypto v0.38.0
	golang.org/x/sync v0.14.0
	golang.org/x/sys v0.33.0
	golang.org/x/text v0.25.0
	golang.org/x/tools v0.33.0
	google.golang.org/genproto/googleapis/api v0.0.0-20250519155744-55703ea1f237
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250519155744-55703ea1f237
	google.golang.org/grpc v1.72.1
	google.golang.org/protobuf v1.36.6
	gopkg.in/reform.v1 v1.5.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.30.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.19 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/charmbracelet/colorprofile v0.2.3-0.20250311203215-f60798e515dc // indirect
	github.com/charmbracelet/x/ansi v0.8.0 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13-0.20250311204145-2c3ea96c31dd // indirect
	github.com/charmbracelet/x/term v0.2.1 // indirect
	github.com/dennwc/varint v1.0.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/googleapis v0.0.0-20180223154316-0cd9801be74a // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/grafana/regexp v0.0.0-20240518133315-a468a5bfb3bc // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-msgpack/v2 v2.1.1 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/miekg/dns v1.1.65 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mwitkow/go-proto-validators v0.3.2 // indirect
	github.com/posener/complete v1.2.3 // indirect
	github.com/riywo/loginshell v0.0.0-20200815045211-7d26008be1ab // indirect
	github.com/sean-/seed v0.0.0-20170313163322-e2103e2c3529 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.36.0 // indirect
	go.opentelemetry.io/otel/metric v1.36.0 // indirect
	go.opentelemetry.io/otel/sdk v1.36.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.18.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.10.0
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph v0.9.0
	github.com/AzureAD/microsoft-authentication-library-for-go v1.4.2 // indirect
	github.com/ClickHouse/ch-go v0.66.0 // indirect
	github.com/HdrHistogram/hdrhistogram-go v1.1.2
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/charmbracelet/harmonica v0.2.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.7.1 // indirect
	github.com/go-openapi/analysis v0.23.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/loads v0.22.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-msgpack v1.1.5 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-sockaddr v1.0.6 // indirect
	github.com/hashicorp/golang-lru v0.6.0 // indirect
	github.com/hashicorp/memberlist v0.5.0
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/paulmach/orb v0.11.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.opentelemetry.io/otel v1.36.0 // indirect
	go.opentelemetry.io/otel/trace v1.36.0 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	gotest.tools/v3 v3.3.0 // indirect
)
