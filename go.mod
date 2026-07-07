module github.com/percona/pmm

go 1.26.4

replace github.com/go-openapi/spec => github.com/Percona-Lab/spec v0.0.0-20260107142235-15cbcf569b9f

replace github.com/alecthomas/kingpin/v2 => github.com/Percona-Lab/kingpin/v2 v2.0.0-20251219125400-95ff74a47c0a

require (
	github.com/AlekSi/pointer v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.14.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph v0.10.0
	github.com/ClickHouse/clickhouse-go/v2 v2.47.0
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/alecthomas/kingpin/v2 v2.4.0
	github.com/alecthomas/kong v1.15.0
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b
	github.com/aws/aws-sdk-go-v2 v1.42.0
	github.com/aws/aws-sdk-go-v2/config v1.32.25
	github.com/aws/aws-sdk-go-v2/credentials v1.19.24
	github.com/aws/aws-sdk-go-v2/service/rds v1.119.3
	github.com/aws/smithy-go v1.27.2
	github.com/blang/semver v3.5.1+incompatible
	github.com/brianvoe/gofakeit/v6 v6.28.0
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/envoyproxy/protoc-gen-validate v1.3.3
	github.com/go-co-op/gocron v1.37.0
	github.com/go-faster/errors v0.7.1
	github.com/go-openapi/errors v0.22.8
	github.com/go-openapi/runtime v0.32.1
	github.com/go-openapi/strfmt v0.26.3
	github.com/go-openapi/swag v0.27.0
	github.com/go-openapi/validate v0.26.0
	github.com/go-sql-driver/mysql v1.10.0
	github.com/gogo/status v1.1.1
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/gomodule/redigo v1.9.2
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	github.com/grafana/grafana-api-golang-client v0.27.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0
	github.com/hashicorp/go-version v1.9.0
	github.com/hashicorp/memberlist v0.5.0
	github.com/hashicorp/raft v1.7.3
	github.com/hashicorp/raft-boltdb/v2 v2.3.1
	github.com/jmoiron/sqlx v1.4.0
	github.com/jotaen/kong-completion v0.0.5
	github.com/lib/pq v1.12.3
	github.com/minio/minio-go/v7 v7.2.0
	github.com/percona/exporter_shared v0.7.5
	github.com/percona/go-mysql v0.0.0-20251202083530-b3e1c16efc74
	github.com/percona/percona-toolkit v0.0.0-20251217124347-6b3e5ed485dd
	github.com/percona/platform v0.0.0-20260424191126-0d2cfd8ceb51
	github.com/percona/promconfig v0.3.0
	github.com/pganalyze/pg_query_go/v6 v6.2.2
	github.com/pkg/sftp v1.13.6
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/common v0.69.0
	github.com/prometheus/prometheus v0.313.0
	github.com/ramr/go-reaper v0.3.1
	github.com/robfig/cron/v3 v3.0.2-0.20210106135023-bc59245fe10e
	github.com/sirupsen/logrus v1.9.4
	github.com/stretchr/objx v0.5.3
	github.com/stretchr/testify v1.11.1
	github.com/tink-crypto/tink-go v0.0.0-20230613075026-d6de17e3f164
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78
	go.mongodb.org/mongo-driver v1.17.7
	go.starlark.net v0.0.0-20230717150657-8a3343210976
	go.yaml.in/yaml/v3 v3.0.4
	golang.org/x/crypto v0.53.0
	golang.org/x/sync v0.21.0
	golang.org/x/sys v0.46.0
	golang.org/x/text v0.38.0
	golang.org/x/tools v0.47.0
	google.golang.org/genproto/googleapis/api v0.0.0-20260615183401-62b3387ff324
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260610212136-7ab31c22f7ad
	google.golang.org/grpc v1.82.0
	google.golang.org/protobuf v1.36.11
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22
	gopkg.in/reform.v1 v1.5.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go v1.36.11-20250718181942-e35f9b667443.1 // indirect
	buf.build/gen/go/bufbuild/protodescriptor/protocolbuffers/go v1.36.11-20250109164928-1da0de137947.1 // indirect
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260415201107-50325440f8f2.1 // indirect
	buf.build/gen/go/bufbuild/registry/connectrpc/go v1.19.2-20260126144947-819582968857.1 // indirect
	buf.build/gen/go/bufbuild/registry/protocolbuffers/go v1.36.11-20260126144947-819582968857.1 // indirect
	buf.build/gen/go/pluginrpc/pluginrpc/protocolbuffers/go v1.36.11-20241007202033-cf42259fcbfc.1 // indirect
	buf.build/go/app v0.2.1-0.20260407195847-833f8f978cda // indirect
	buf.build/go/bufplugin v0.10.0 // indirect
	buf.build/go/bufprivateusage v0.1.0 // indirect
	buf.build/go/interrupt v1.1.0 // indirect
	buf.build/go/protovalidate v1.2.0 // indirect
	buf.build/go/protoyaml v0.6.0 // indirect
	buf.build/go/spdx v0.2.0 // indirect
	buf.build/go/standard v0.1.1-0.20260325175353-2b287e071df5 // indirect
	cel.dev/expr v0.25.1 // indirect
	connectrpc.com/connect v1.19.2 // indirect
	connectrpc.com/otelconnect v0.9.0 // indirect
	dario.cat/mergo v1.0.2 // indirect
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.22.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.12.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.7.2 // indirect
	github.com/BurntSushi/go-sumtype v0.0.0-20250321152530-ada76606277d // indirect
	github.com/ClickHouse/ch-go v0.73.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/Masterminds/sprig/v3 v3.3.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProtonMail/go-crypto v1.1.6 // indirect
	github.com/aclements/go-moremath v0.0.0-20210112150236-f10218a38794 // indirect
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/apache/skywalking-eyes v0.8.0 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.30 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.29 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.2.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.31.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.36.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.43.3 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bmatcuk/doublestar/v2 v2.0.4 // indirect
	github.com/boltdb/bolt v1.3.1 // indirect
	github.com/bradleyfalzon/ghinstallation v0.1.2 // indirect
	github.com/bufbuild/buf v1.69.0 // indirect
	github.com/bufbuild/protocompile v0.14.2-0.20260429155904-12ef1ef2ce91 // indirect
	github.com/bufbuild/protoplugin v0.0.0-20260414125817-25d1d281b46b // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chigopher/pathlib v0.19.1 // indirect
	github.com/cilium/ebpf v0.11.0 // indirect
	github.com/cli/browser v1.3.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/cloudflare/circl v1.6.3 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.18.2 // indirect
	github.com/cosiner/argv v0.1.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/cyphar/filepath-securejoin v0.6.1 // indirect
	github.com/daixiang0/gci v0.14.0 // indirect
	github.com/denisenkom/go-mssqldb v0.12.3 // indirect
	github.com/dennwc/varint v1.0.0 // indirect
	github.com/derekparker/trie v0.0.0-20221213183930-4c74548207f4 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/cli v29.4.1+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.9.6 // indirect
	github.com/docker/go-connections v0.7.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/gliderlabs/ssh v0.3.9-0.20241212082318-d137aad99cd6 // indirect
	github.com/go-delve/delve v1.21.2 // indirect
	github.com/go-delve/liner v1.2.3-0.20231231155935-4726ab1d7f62 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.9.0 // indirect
	github.com/go-git/go-git/v5 v5.19.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/analysis v0.25.2 // indirect
	github.com/go-openapi/inflect v0.21.5 // indirect
	github.com/go-openapi/jsonpointer v0.24.0 // indirect
	github.com/go-openapi/jsonreference v0.21.6 // indirect
	github.com/go-openapi/loads v0.23.4 // indirect
	github.com/go-openapi/runtime/server-middleware v0.30.0 // indirect
	github.com/go-openapi/spec v0.22.5 // indirect
	github.com/go-openapi/swag/cmdutils v0.27.0 // indirect
	github.com/go-openapi/swag/conv v0.27.0 // indirect
	github.com/go-openapi/swag/fileutils v0.27.0 // indirect
	github.com/go-openapi/swag/jsonname v0.26.0 // indirect
	github.com/go-openapi/swag/jsonutils v0.27.0 // indirect
	github.com/go-openapi/swag/loading v0.27.0 // indirect
	github.com/go-openapi/swag/mangling v0.27.0 // indirect
	github.com/go-openapi/swag/netutils v0.27.0 // indirect
	github.com/go-openapi/swag/stringutils v0.27.0 // indirect
	github.com/go-openapi/swag/typeutils v0.27.0 // indirect
	github.com/go-openapi/swag/yamlutils v0.27.0 // indirect
	github.com/go-swagger/go-swagger v0.33.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/gofrs/flock v0.13.0 // indirect
	github.com/gogo/googleapis v0.0.0-20180223154316-0cd9801be74a // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/cel-go v0.28.0 // indirect
	github.com/google/go-containerregistry v0.21.5 // indirect
	github.com/google/go-dap v0.12.0 // indirect
	github.com/google/go-github/v28 v28.1.1 // indirect
	github.com/google/go-github/v33 v33.0.0 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/google/licensecheck v0.3.1 // indirect
	github.com/gorilla/handlers v1.5.2 // indirect
	github.com/grafana/regexp v0.0.0-20250905093917-f7b3be9d1853 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-metrics v0.5.4 // indirect
	github.com/hashicorp/go-msgpack v1.1.5 // indirect
	github.com/hashicorp/go-msgpack/v2 v2.1.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-sockaddr v1.0.6 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/golang-lru v0.6.0 // indirect
	github.com/hexops/gotextdiff v1.0.3 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgx v3.6.2+incompatible // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jdx/go-netrc v1.0.0 // indirect
	github.com/jessevdk/go-flags v1.6.1 // indirect
	github.com/jinzhu/copier v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jstemmer/go-junit-report/v2 v2.1.0 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/klauspost/crc32 v1.3.0 // indirect
	github.com/klauspost/pgzip v1.2.6 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lyft/protoc-gen-star/v2 v2.0.4 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/mattn/go-runewidth v0.0.23 // indirect
	github.com/mattn/go-shellwords v1.0.12 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/miekg/dns v1.1.72 // indirect
	github.com/minio/crc64nvme v1.1.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/moby/api v1.55.0 // indirect
	github.com/moby/moby/client v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mwitkow/go-proto-validators v0.3.2 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/onsi/gomega v1.39.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/paulmach/orb v0.13.0 // indirect
	github.com/pelletier/go-toml/v2 v2.3.1 // indirect
	github.com/petermattis/goid v0.0.0-20260330135022-df67b199bc81 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.27 // indirect
	github.com/pjbgf/sha1cd v0.6.0 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/posener/complete v1.2.3 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.59.1 // indirect
	github.com/reviewdog/errorformat v0.0.0-20250320004447-223c26dbe212 // indirect
	github.com/reviewdog/reviewdog v0.9.13 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/riywo/loginshell v0.0.0-20200815045211-7d26008be1ab // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/rs/cors v1.11.1 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/sean-/seed v0.0.0-20170313163322-e2103e2c3529 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/skeema/knownhosts v1.3.1 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/spf13/viper v1.21.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tetratelabs/wazero v1.11.0 // indirect
	github.com/tidwall/btree v1.8.1 // indirect
	github.com/tinylib/msgp v1.6.1 // indirect
	github.com/toqueteos/webbrowser v1.2.1 // indirect
	github.com/vbatts/tar-split v0.12.3 // indirect
	github.com/vektra/mockery/v2 v2.53.6 // indirect
	github.com/xanzy/go-gitlab v0.20.1 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.2.0 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.etcd.io/bbolt v1.3.5 // indirect
	go.lsp.dev/jsonrpc2 v0.10.0 // indirect
	go.lsp.dev/pkg v0.0.0-20210717090340-384b27a52fb2 // indirect
	go.lsp.dev/protocol v0.12.0 // indirect
	go.lsp.dev/uri v0.3.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.69.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/mock v0.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	golang.org/x/arch v0.11.0 // indirect
	golang.org/x/exp v0.0.0-20260410095643-746e56fc9e2f // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/perf v0.0.0-20260409210113-8e83ce0f7b1c // indirect
	golang.org/x/telemetry v0.0.0-20260625142307-59b4966ccb57 // indirect
	golang.org/x/term v0.44.0 // indirect
	google.golang.org/appengine v1.6.8-0.20221117013220-504804fb50de // indirect
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.6.0 // indirect
	gopkg.in/ini.v1 v1.67.2 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	mvdan.cc/gofumpt v0.10.0 // indirect
	mvdan.cc/xurls/v2 v2.6.0 // indirect
	pluginrpc.com/pluginrpc v0.5.0 // indirect
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
	github.com/jstemmer/go-junit-report/v2
	github.com/reviewdog/reviewdog/cmd/reviewdog
	github.com/vektra/mockery/v2
	golang.org/x/perf/cmd/benchstat
	golang.org/x/tools/cmd/goimports
	golang.org/x/tools/cmd/stringer
	google.golang.org/grpc/cmd/protoc-gen-go-grpc
	google.golang.org/protobuf/cmd/protoc-gen-go
	gopkg.in/reform.v1/reform
	gopkg.in/reform.v1/reform-db
	mvdan.cc/gofumpt
)
