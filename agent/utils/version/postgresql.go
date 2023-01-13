// Copyright 2019 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package version

import (
	"context"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

// regexps to extract version numbers from the `SELECT version()` output
var (
	postgresDBRegexp = regexp.MustCompile(`PostgreSQL (\d+\.?\d+)`)
)

func ParsePostgreSQLVersion(v string) string {
	m := postgresDBRegexp.FindStringSubmatch(v)
	if len(m) != 2 {
		return ""
	}

	return m[1]
}

const (
	postgresVersionQuery = `SELECT version()`
)

// PostgreSQLVersion represent major, minor numbers of PostgreSQL version separated by comma.
type PostgreSQLVersion struct {
	text   string
	number float64
}

// GetPostgreSQLVersion returns MAJOR.MINOR PostgreSQL version (e.g. "5.6", "8.0", etc.).
func GetPostgreSQLVersion(ctx context.Context, q reform.DBTXContext) (PostgreSQLVersion, error) {
	var version string
	err := q.QueryRowContext(ctx, postgresVersionQuery).Scan(&version)
	if err != nil {
		return PostgreSQLVersion{}, err
	}

	text := postgresDBRegexp.FindStringSubmatch(version)
	if len(text) < 2 {
		return PostgreSQLVersion{}, errors.New("postgresql version not found")
	}

	number, err := strconv.ParseFloat(text[1], 64)
	if err != nil {
		return PostgreSQLVersion{}, err
	}

	return PostgreSQLVersion{text: text[1], number: number}, nil
}

// Float represent PostgreSQL version in float format.
func (v PostgreSQLVersion) Float() float64 {
	return v.number
}

func (v PostgreSQLVersion) String() string {
	return v.text
}
