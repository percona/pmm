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

package version

import (
	"regexp"
	"strings"
)

// regexps to extract version numbers from the `SELECT version()` output.
var (
	postgresDBRegexp = regexp.MustCompile(`PostgreSQL ([\d\.]+)`)
)

// ParsePostgreSQLVersion parses the given PostgreSQL version string (such as the
// output of `SELECT version()`) and returns the major version and the second
// numeric component as strings.
//
// For PostgreSQL versions prior to 10, this typically corresponds to
// "major" and "minor" (e.g., 9.6.5 -> "9", "6"). For PostgreSQL 10 and above,
// PostgreSQL uses a two-part versioning scheme where the second component is
// the patch level (e.g., 18.2 -> "18", "2"), not a traditional minor version.
func ParsePostgreSQLVersion(v string) (string, string) {
	m := postgresDBRegexp.FindStringSubmatch(v)
	if len(m) != 2 {
		return "", ""
	}

	parts := strings.Split(m[1], ".")
	if len(parts) == 1 {
		return parts[0], ""
	}

	return parts[0], parts[1]
}
