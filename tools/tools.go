// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

//go:build tools

package tools

import (
	_ "github.com/BurntSushi/go-sumtype"
	_ "github.com/Percona-Lab/swagger-order"
	_ "github.com/alta/protopatch/cmd/protoc-gen-go-patch"
	_ "github.com/apache/skywalking-eyes/cmd/license-eye"
	_ "github.com/bufbuild/buf/cmd/buf"
	_ "github.com/daixiang0/gci"
	_ "github.com/envoyproxy/protoc-gen-validate"
	_ "github.com/go-delve/delve/cmd/dlv"
	_ "github.com/go-openapi/runtime/client"
	_ "github.com/go-openapi/spec"
	_ "github.com/go-swagger/go-swagger/cmd/swagger"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"
	_ "github.com/jstemmer/go-junit-report"
	_ "github.com/quasilyte/go-consistent"
	_ "github.com/reviewdog/reviewdog/cmd/reviewdog"
	_ "github.com/vburenin/ifacemaker"
	_ "github.com/vektra/mockery/v2"
	_ "golang.org/x/perf/cmd/benchstat"
	_ "golang.org/x/tools/cmd/goimports"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
	_ "gopkg.in/reform.v1/reform"
	_ "gopkg.in/reform.v1/reform-db"
	_ "mvdan.cc/gofumpt"
)

//go:generate go build -o ../bin/benchstat golang.org/x/perf/cmd/benchstat
//go:generate go build -o ../bin/buf github.com/bufbuild/buf/cmd/buf
//go:generate go build -o ../bin/dlv github.com/go-delve/delve/cmd/dlv
//go:generate go build -o ../bin/gci github.com/daixiang0/gci
//go:generate go build -o ../bin/go-consistent github.com/quasilyte/go-consistent
//go:generate go build -o ../bin/go-junit-report github.com/jstemmer/go-junit-report
//go:generate go build -o ../bin/go-sumtype github.com/BurntSushi/go-sumtype
//go:generate go build -o ../bin/gofumpt mvdan.cc/gofumpt
//go:generate go build -o ../bin/goimports golang.org/x/tools/cmd/goimports
//go:generate go build -o ../bin/mockery github.com/vektra/mockery/v2
//go:generate go build -o ../bin/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
//go:generate go build -o ../bin/protoc-gen-go-grpc google.golang.org/grpc/cmd/protoc-gen-go-grpc
//go:generate go build -o ../bin/protoc-gen-go-patch github.com/alta/protopatch/cmd/protoc-gen-go-patch
//go:generate go build -o ../bin/protoc-gen-grpc-gateway github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway
//go:generate go build -o ../bin/protoc-gen-openapiv2 github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2
//go:generate go build -o ../bin/reform gopkg.in/reform.v1/reform
//go:generate go build -o ../bin/reviewdog github.com/reviewdog/reviewdog/cmd/reviewdog
//go:generate go build -o ../bin/swagger github.com/go-swagger/go-swagger/cmd/swagger
//go:generate go build -o ../bin/swagger-order github.com/Percona-Lab/swagger-order
//go:generate go build -o ../bin/protoc-gen-validate github.com/envoyproxy/protoc-gen-validate
//go:generate go build -o ../bin/ifacemaker github.com/vburenin/ifacemaker
//go:generate env CGO_ENABLED=0 go build -o ../bin/license-eye github.com/apache/skywalking-eyes/cmd/license-eye
