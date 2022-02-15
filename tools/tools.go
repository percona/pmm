// pmm-managed
// Copyright (C) 2017 Percona LLC
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
// +build tools

package tools

import (
	_ "github.com/BurntSushi/go-sumtype"
	_ "github.com/go-delve/delve/cmd/dlv"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/kevinburke/go-bindata/go-bindata"
	_ "github.com/quasilyte/go-consistent"
	_ "github.com/reviewdog/reviewdog/cmd/reviewdog"
	_ "github.com/vektra/mockery/cmd/mockery"
	_ "golang.org/x/perf/cmd/benchstat"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/tools/gopls"
	_ "gopkg.in/reform.v1/reform"
	_ "gopkg.in/reform.v1/reform-db"
	_ "mvdan.cc/gofumpt"
)

//go:generate go build -o ../bin/benchstat golang.org/x/perf/cmd/benchstat
//go:generate go build -o ../bin/dlv github.com/go-delve/delve/cmd/dlv
//go:generate go build -o ../bin/go-bindata github.com/kevinburke/go-bindata/go-bindata
//go:generate go build -o ../bin/go-sumtype github.com/BurntSushi/go-sumtype
//go:generate go build -o ../bin/gofumpt mvdan.cc/gofumpt
//go:generate go build -o ../bin/goimports golang.org/x/tools/cmd/goimports
//go:generate go build -o ../bin/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint
//go:generate go build -o ../bin/gopls golang.org/x/tools/gopls
//go:generate go build -o ../bin/mockery github.com/vektra/mockery/cmd/mockery
//go:generate go build -o ../bin/reform gopkg.in/reform.v1/reform
//go:generate go build -o ../bin/reviewdog github.com/reviewdog/reviewdog/cmd/reviewdog
//go:generate go build -o ../bin/go-consistent github.com/quasilyte/go-consistent
