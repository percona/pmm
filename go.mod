module github.com/percona/pmm

go 1.21

// Update saas with
// go get -v github.com/percona-platform/saas@latest

replace github.com/grpc-ecosystem/go-grpc-prometheus => github.com/Percona-Lab/go-grpc-prometheus v0.0.0-20230116133345-3487748d4592

replace github.com/go-openapi/spec => github.com/Percona-Lab/spec v0.20.5-percona

replace gopkg.in/alecthomas/kingpin.v2 => github.com/Percona-Lab/kingpin v2.2.6-percona+incompatible

replace golang.org/x/crypto => github.com/percona-lab/crypto v0.0.0-20231108144114-756dfb24eaf2

replace github.com/ClickHouse/clickhouse-go/151 => github.com/ClickHouse/clickhouse-go v1.5.1 // clickhouse-go/v2 cannot work with 1.5.1 which we need for QAN-API

require (
	github.com/AlekSi/pointer v1.2.0
	github.com/ClickHouse/clickhouse-go/151 v0.0.0-00010101000000-000000000000
	github.com/ClickHouse/clickhouse-go/v2 v2.15.0
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/alecthomas/kong v0.8.0
	github.com/alecthomas/units v0.0.0-20211218093645-b94a6e3cc137
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2
	github.com/aws/aws-sdk-go v1.49.1
	github.com/blang/semver v3.5.1+incompatible
	github.com/brianvoe/gofakeit/v6 v6.26.0
	github.com/charmbracelet/bubbles v0.16.1
	github.com/charmbracelet/bubbletea v0.24.1
	github.com/charmbracelet/lipgloss v0.9.0
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/docker v24.0.6+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/envoyproxy/protoc-gen-validate v1.0.2
	github.com/go-co-op/gocron v1.36.0
	github.com/go-openapi/errors v0.20.4
	github.com/go-openapi/runtime v0.26.0
	github.com/go-openapi/strfmt v0.21.7
	github.com/go-openapi/swag v0.22.3
	github.com/go-openapi/validate v0.22.1
	github.com/go-sql-driver/mysql v1.7.1
	github.com/golang-migrate/migrate/v4 v4.16.1
	github.com/golang/protobuf v1.5.3
	github.com/google/uuid v1.4.0
	github.com/grafana/grafana-api-golang-client v0.27.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.0
	github.com/hashicorp/go-version v1.6.0
	github.com/hashicorp/raft v1.6.0
	github.com/jmoiron/sqlx v1.3.5
	github.com/jotaen/kong-completion v0.0.5
	github.com/lib/pq v1.10.9
	github.com/minio/minio-go/v7 v7.0.55
	github.com/operator-framework/api v0.17.6
	github.com/operator-framework/operator-lifecycle-manager v0.24.0
	github.com/percona-platform/dbaas-api v0.0.0-20230103182808-d79c449a9f4c
	github.com/percona-platform/saas v0.0.0-20230728161159-ad6bdeb8a3d9
	github.com/percona/dbaas-operator v0.1.6
	github.com/percona/exporter_shared v0.7.4
	github.com/percona/go-mysql v0.0.0-20210427141028-73d29c6da78c
	github.com/percona/percona-toolkit v3.2.1+incompatible
	github.com/percona/promconfig v0.2.5
	github.com/pganalyze/pg_query_go/v2 v2.2.0
	github.com/pkg/errors v0.9.1
	github.com/pkg/sftp v1.13.6
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/alertmanager v0.26.0
	github.com/prometheus/client_golang v1.17.0
	github.com/prometheus/common v0.45.0
	github.com/ramr/go-reaper v0.2.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/objx v0.5.0
	github.com/stretchr/testify v1.8.4
	go.mongodb.org/mongo-driver v1.13.1
	go.starlark.net v0.0.0-20230717150657-8a3343210976
	golang.org/x/crypto v0.16.0
	golang.org/x/sync v0.5.0
	golang.org/x/sys v0.15.0
	golang.org/x/text v0.14.0
	golang.org/x/tools v0.16.0
	google.golang.org/genproto/googleapis/api v0.0.0-20231002182017-d307bd883b97
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231002182017-d307bd883b97
	google.golang.org/grpc v1.60.1
	google.golang.org/protobuf v1.31.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/reform.v1 v1.5.1
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.28.0
	k8s.io/apiextensions-apiserver v0.26.2
	k8s.io/apimachinery v0.28.0
	k8s.io/cli-runtime v0.28.0
	k8s.io/client-go v0.28.0
	k8s.io/kubectl v0.28.0
	sigs.k8s.io/controller-runtime v0.14.6
)

require (
	github.com/Azure/azure-pipeline-go v0.2.3 // indirect
	github.com/Azure/azure-storage-blob-go v0.14.0 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/benbjohnson/clock v1.3.5 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/emicklei/go-restful/v3 v3.9.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.0.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20221118152302-e6195bd50e26 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/hashicorp/go-hclog v1.5.0 // indirect
	github.com/hashicorp/go-msgpack/v2 v2.1.1 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.2 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-ieproxy v0.0.1 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/minio/minio-go v6.0.14+incompatible // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mwitkow/go-proto-validators v0.3.2 // indirect
	github.com/percona/percona-backup-mongodb v1.8.1 // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/posener/complete v1.2.3 // indirect
	github.com/riywo/loginshell v0.0.0-20200815045211-7d26008be1ab // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	go.opentelemetry.io/otel/metric v1.19.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	google.golang.org/genproto v0.0.0-20231002182017-d307bd883b97 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	k8s.io/klog/v2 v2.100.1 // indirect
	k8s.io/kube-openapi v0.0.0-20230717233707-2695361300d9 // indirect
	k8s.io/utils v0.0.0-20230406110748-d93618cff8a2 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/kustomize/api v0.13.5-0.20230601165947-6ce0bf390ce3 // indirect
	sigs.k8s.io/kustomize/kyaml v0.14.3-0.20230601165947-6ce0bf390ce3 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.9.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.4.0
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.5.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph v0.9.0
	github.com/AzureAD/microsoft-authentication-library-for-go v1.1.1 // indirect
	github.com/ClickHouse/ch-go v0.58.2 // indirect
	github.com/ClickHouse/clickhouse-go v1.5.4 // indirect
	github.com/HdrHistogram/hdrhistogram-go v1.1.2
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/andybalholm/brotli v1.0.6 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/charmbracelet/harmonica v0.2.0 // indirect
	github.com/cloudflare/golz4 v0.0.0-20150217214814-ef862a3cdc58 // indirect
	github.com/containerd/console v1.0.4-0.20230313162750-1ae8d489ac81 // indirect
	github.com/docker/distribution v2.8.2+incompatible // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.6.1 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-openapi/analysis v0.21.4 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/loads v0.21.2 // indirect
	github.com/go-openapi/spec v0.20.8 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-msgpack v1.1.5 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.6.0 // indirect
	github.com/hashicorp/memberlist v0.5.0
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.18 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/miekg/dns v1.1.50 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/montanaflynn/stats v0.7.0 // indirect
	github.com/muesli/ansi v0.0.0-20211018074035-2e021307bc4b // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.15.2 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc4 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/paulmach/orb v0.10.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.18 // indirect
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8 // indirect
	github.com/prometheus/client_model v0.4.1-0.20230718164431-9a2bf3000d16 // indirect
	github.com/prometheus/common/sigv4 v0.1.0 // indirect
	github.com/prometheus/exporter-toolkit v0.10.0 // indirect
	github.com/prometheus/procfs v0.11.1 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/sean-/seed v0.0.0-20170313163322-e2103e2c3529 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/shurcooL/httpfs v0.0.0-20190707220628-8d4bc4ba7749 // indirect
	github.com/shurcooL/vfsgen v0.0.0-20200824052919-0d455de96546 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	go.opentelemetry.io/otel v1.19.0 // indirect
	go.opentelemetry.io/otel/trace v1.19.0 // indirect
	golang.org/x/mod v0.14.0 // indirect
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/oauth2 v0.13.0 // indirect
	golang.org/x/term v0.15.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gotest.tools/v3 v3.3.0 // indirect
)
