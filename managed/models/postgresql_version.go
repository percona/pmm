// Copyright (C) 2024 Percona LLC
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

package models

import (
	"context"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

// regexps to extract version numbers from the `SELECT version()` output.
var (
	postgresDBRegexp = regexp.MustCompile(`PostgreSQL (\d+\.?\d+)`)
)

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
		return PostgreSQLVersion{text: text[1]}, err
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
