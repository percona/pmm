module github.com/percona/pmm

go 1.24.9

// Update saas with
// go get -v github.com/percona/saas@latest

replace github.com/grpc-ecosystem/go-grpc-prometheus => github.com/Percona-Lab/go-grpc-prometheus v0.0.0-20230116133345-3487748d4592

replace github.com/go-openapi/spec => github.com/Percona-Lab/spec v0.21.0-percona

replace github.com/alecthomas/kingpin/v2 => github.com/Percona-Lab/kingpin/v2 v2.4.0-percona

replace golang.org/x/crypto => github.com/percona-lab/crypto v0.0.0-20231108144114-756dfb24eaf2

require (
	github.com/AlekSi/pointer v1.2.0
	github.com/ClickHouse/clickhouse-go/v2 v2.40.1
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/alecthomas/kingpin/v2 v2.4.0
	github.com/alecthomas/kong v1.12.0
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b
	github.com/aws/aws-sdk-go-v2 v1.39.2
	github.com/aws/aws-sdk-go-v2/config v1.31.12
	github.com/aws/aws-sdk-go-v2/credentials v1.18.16
	github.com/aws/aws-sdk-go-v2/service/rds v1.108.1
	github.com/aws/smithy-go v1.23.0
	github.com/blang/semver v3.5.1+incompatible
	github.com/brianvoe/gofakeit/v6 v6.28.0
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/envoyproxy/protoc-gen-validate v1.2.1
	github.com/go-co-op/gocron v1.37.0
	github.com/go-openapi/errors v0.22.4
	github.com/go-openapi/runtime v0.29.0
	github.com/go-openapi/strfmt v0.25.0
	github.com/go-openapi/swag v0.25.1
	github.com/go-openapi/validate v0.25.0
	github.com/go-sql-driver/mysql v1.9.0
	github.com/gogo/status v1.1.1
	github.com/golang-migrate/migrate/v4 v4.18.1
	github.com/gomodule/redigo v1.9.2
	github.com/google/tink/go v1.7.0
	github.com/google/uuid v1.6.0
	github.com/grafana/grafana-api-golang-client v0.27.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2
	github.com/hashicorp/go-version v1.7.0
	github.com/hashicorp/raft v1.7.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/jotaen/kong-completion v0.0.5
	github.com/lib/pq v1.10.9
	github.com/minio/minio-go/v7 v7.0.55
	github.com/percona/exporter_shared v0.7.5
	github.com/percona/go-mysql v0.0.0-20250429063950-bc274ddba990
	github.com/percona/percona-toolkit v0.0.0-20250623101308-06d5966106f5
	github.com/percona/promconfig v0.2.5
	github.com/percona/saas v0.0.0-20240923141535-da19f6682c6e
	github.com/pganalyze/pg_query_go/v6 v6.1.0
	github.com/pkg/errors v0.9.1
	github.com/pkg/sftp v1.13.6
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/common v0.67.1
	github.com/prometheus/prometheus v0.307.3
	github.com/ramr/go-reaper v0.3.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/objx v0.5.2
	github.com/stretchr/testify v1.11.1
	github.com/tink-crypto/tink-go v0.0.0-20230613075026-d6de17e3f164
	go.mongodb.org/mongo-driver v1.17.6
	go.starlark.net v0.0.0-20230717150657-8a3343210976
	golang.org/x/crypto v0.43.0
	golang.org/x/sync v0.18.0
	golang.org/x/sys v0.37.0
	golang.org/x/text v0.31.0
	golang.org/x/tools v0.38.0
	google.golang.org/genproto/googleapis/api v0.0.0-20250929231259-57b25ae835d4
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250922171735-9219d122eba9
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22
	gopkg.in/reform.v1 v1.5.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go v1.36.6-20250718181942-e35f9b667443.1 // indirect
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.6-20250912141014-52f32327d4b0.1 // indirect
	buf.build/gen/go/bufbuild/registry/connectrpc/go v1.18.1-20250819211657-a3dd0d3ea69b.1 // indirect
	buf.build/gen/go/bufbuild/registry/protocolbuffers/go v1.36.6-20250924144421-cb55f06efbd2.1 // indirect
	buf.build/gen/go/pluginrpc/pluginrpc/protocolbuffers/go v1.36.6-20241007202033-cf42259fcbfc.1 // indirect
	buf.build/go/app v0.1.0 // indirect
	buf.build/go/bufplugin v0.9.0 // indirect
	buf.build/go/interrupt v1.1.0 // indirect
	buf.build/go/protovalidate v0.14.0 // indirect
	buf.build/go/protoyaml v0.6.0 // indirect
	buf.build/go/spdx v0.2.0 // indirect
	buf.build/go/standard v0.1.0 // indirect
	cel.dev/expr v0.24.0 // indirect
	cloud.google.com/go v0.121.2 // indirect
	cloud.google.com/go/auth v0.16.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.7.0 // indirect
	cloud.google.com/go/datastore v1.20.0 // indirect
	connectrpc.com/connect v1.18.1 // indirect
	connectrpc.com/otelconnect v0.7.2 // indirect
	dario.cat/mergo v1.0.1 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.29.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.38.6 // indirect
	github.com/dennwc/varint v1.0.0 // indirect
	github.com/derekparker/trie v0.0.0-20230829180723-39f4de51ef7d // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/cli v28.3.3+incompatible // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker v28.3.3+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.9.3 // indirect
	github.com/docker/go-connections v0.6.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/go-chi/chi/v5 v5.2.2 // indirect
	github.com/go-delve/delve v1.21.2 // indirect
	github.com/go-delve/liner v1.2.3-0.20231231155935-4726ab1d7f62 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.6.0 // indirect
	github.com/go-git/go-git/v5 v5.13.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/swag/cmdutils v0.25.1 // indirect
	github.com/go-openapi/swag/conv v0.25.1 // indirect
	github.com/go-openapi/swag/fileutils v0.25.1 // indirect
	github.com/go-openapi/swag/jsonname v0.25.1 // indirect
	github.com/go-openapi/swag/jsonutils v0.25.1 // indirect
	github.com/go-openapi/swag/loading v0.25.1 // indirect
	github.com/go-openapi/swag/mangling v0.25.1 // indirect
	github.com/go-openapi/swag/netutils v0.25.1 // indirect
	github.com/go-openapi/swag/stringutils v0.25.1 // indirect
	github.com/go-openapi/swag/typeutils v0.25.1 // indirect
	github.com/go-openapi/swag/yamlutils v0.25.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gogo/googleapis v0.0.0-20180223154316-0cd9801be74a // indirect
	github.com/golang-jwt/jwt/v4 v4.4.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
	github.com/golang-sql/civil v0.0.0-20190719163853-cb61b32ac6fe // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/grafana/regexp v0.0.0-20250905093917-f7b3be9d1853 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-msgpack/v2 v2.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/haya14busa/go-actions-toolkit v0.0.0-20200105081403-ca0307860f01 // indirect
	github.com/hexops/gotextdiff v1.0.3 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgx v3.6.2+incompatible // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jdx/go-netrc v1.0.0 // indirect
	github.com/jessevdk/go-flags v1.5.0 // indirect
	github.com/jinzhu/copier v0.4.0 // indirect
	github.com/jstemmer/go-junit-report v1.0.0 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/kisielk/gotool v1.0.0 // indirect
	github.com/klauspost/pgzip v1.2.6 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lyft/protoc-gen-star/v2 v2.0.4-0.20230330145011-496ad1ac90a4 // indirect
	github.com/magiconair/properties v1.8.10 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/miekg/dns v1.1.68 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mwitkow/go-proto-validators v0.3.2 // indirect
	github.com/onsi/gomega v1.37.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/posener/complete v1.2.3 // indirect
	github.com/quasilyte/go-consistent v0.6.0 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/quic-go/quic-go v0.54.0 // indirect
	github.com/reva2/bitbucket-insights-api v1.0.0 // indirect
	github.com/reviewdog/errorformat v0.0.0-20250320004447-223c26dbe212 // indirect
	github.com/reviewdog/go-bitbucket v0.0.0-20201024094602-708c3f6a7de0 // indirect
	github.com/reviewdog/reviewdog v0.14.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/riywo/loginshell v0.0.0-20200815045211-7d26008be1ab // indirect
	github.com/sean-/seed v0.0.0-20170313163322-e2103e2c3529 // indirect
	github.com/segmentio/encoding v0.5.3 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/skeema/knownhosts v1.3.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.14.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/spf13/cobra v1.9.1 // indirect
	github.com/spf13/pflag v1.0.7 // indirect
	github.com/spf13/viper v1.19.0 // indirect
	github.com/stoewer/go-strcase v1.3.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tetratelabs/wazero v1.9.0 // indirect
	github.com/toqueteos/webbrowser v1.2.0 // indirect
	github.com/vbatts/tar-split v0.12.1 // indirect
	github.com/vburenin/ifacemaker v1.2.1 // indirect
	github.com/vektra/mockery/v2 v2.52.1 // indirect
	github.com/vvakame/sdlog v0.0.0-20200409072131-7c0d359efddc // indirect
	github.com/xanzy/go-gitlab v0.114.0 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	mvdan.cc/gofumpt v0.8.0 // indirect
	pluginrpc.com/pluginrpc v0.5.0 // indirect
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.19.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.13.0
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph v0.9.0
	github.com/AzureAD/microsoft-authentication-library-for-go v1.5.0 // indirect
	github.com/ClickHouse/ch-go v0.67.0 // indirect
	github.com/HdrHistogram/hdrhistogram-go v1.1.2
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.7.1 // indirect
	github.com/go-openapi/analysis v0.24.0 // indirect
	github.com/go-openapi/jsonpointer v0.22.1 // indirect
	github.com/go-openapi/jsonreference v0.21.2 // indirect
	github.com/go-openapi/loads v0.23.1 // indirect
	github.com/go-openapi/spec v0.22.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-msgpack v1.1.5 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-sockaddr v1.0.6 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/memberlist v0.5.0
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/paulmach/orb v0.11.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	golang.org/x/mod v0.29.0 // indirect
	golang.org/x/net v0.46.0 // indirect
)

tool (
	github.com/BurntSushi/go-sumtype
	github.com/apache/skywalking-eyes/cmd/license-eye
	github.com/bufbuild/buf/cmd/buf
	github.com/daixiang0/gci
	github.com/envoyproxy/protoc-gen-validate
	github.com/go-delve/delve/cmd/dlv
	github.com/go-swagger/go-swagger/cmd/swagger
	github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway
	github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2
	github.com/jstemmer/go-junit-report
	github.com/quasilyte/go-consistent
	github.com/reviewdog/reviewdog/cmd/reviewdog
	github.com/vburenin/ifacemaker
	github.com/vektra/mockery/v2
	golang.org/x/perf/cmd/benchstat
	golang.org/x/tools/cmd/goimports
	google.golang.org/grpc/cmd/protoc-gen-go-grpc
	google.golang.org/protobuf/cmd/protoc-gen-go
	gopkg.in/reform.v1/reform
	gopkg.in/reform.v1/reform-db
	mvdan.cc/gofumpt
)
