// Copyright (C) 2023 Percona LLC
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

// Package version contains functions for database versions processing.
package version

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/reform.v1"
)

// MySQLVendor represents MySQL vendor.
type MySQLVendor byte

// MySQLVendor represent MySQL vendors.
const (
	OracleVendor MySQLVendor = iota + 1
	PerconaVendor
	MariaDBVendor
)

// MySQLVersion represent major, minor numbers of mysql version separated by comma.
type MySQLVersion struct {
	text   string
	number float64
}

const (
	perconaComment = "percona"
	mariaDBComment = "mariadb"
	debianComment  = "debian"

	mysqlVersionQuery = `SHOW GLOBAL VARIABLES WHERE Variable_name = 'version'`
	commentQuery      = `SHOW GLOBAL VARIABLES WHERE Variable_name = 'version_comment'`
)

var (
	// Regexps to extract version numbers from the `SHOW GLOBAL VARIABLES WHERE Variable_name = 'version'` output.
	mysqlDBRegexp = regexp.MustCompile(`^\d+\.\d+`)
	// Vendors to represent MySQLVendor enum in string format with default value unknown.
	vendors = [...]string{"unknown", "oracle", "percona", "mariadb"}
)

// GetMySQLVersion returns MAJOR.MINOR MySQL version (e.g. "5.6", "8.0", etc.) and vendor.
func GetMySQLVersion(ctx context.Context, q reform.DBTXContext) (MySQLVersion, MySQLVendor, error) {
	var name, version string
	err := q.QueryRowContext(ctx, mysqlVersionQuery).Scan(&name, &version)
	if err != nil {
		return MySQLVersion{}, 0, err
	}
	var ven string
	err = q.QueryRowContext(ctx, commentQuery).Scan(&name, &ven)
	if err != nil {
		return MySQLVersion{}, 0, err
	}

	text := mysqlDBRegexp.FindString(version)
	number, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return MySQLVersion{}, 0, err
	}

	var vendor MySQLVendor
	switch {
	case strings.Contains(strings.ToLower(ven), perconaComment):
		vendor = PerconaVendor
	case strings.Contains(strings.ToLower(ven), mariaDBComment):
		vendor = MariaDBVendor
	case strings.Contains(strings.ToLower(ven), debianComment) && strings.Contains(strings.ToLower(version), mariaDBComment):
		vendor = MariaDBVendor
	default:
		vendor = OracleVendor
	}

	return MySQLVersion{text: text, number: number}, vendor, nil
}

func (v MySQLVendor) String() string {
	if int(v) >= len(vendors) {
		return vendors[0]
	}
	return vendors[v]
}

// Float represent mysql version in float format.
func (v MySQLVersion) Float() float64 {
	return v.number
}

func (v MySQLVersion) String() string {
	return v.text
}
