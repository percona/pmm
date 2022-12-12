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

package queryparser

import (
	"regexp"
	"strings"
	"sync"

	pg_query "github.com/pganalyze/pg_query_go"
	"github.com/pkg/errors"
	"vitess.io/vitess/go/vt/proto/query"
	"vitess.io/vitess/go/vt/sqlparser"
)

var (
	pgRegexp    *regexp.Regexp
	pgRegexpErr error
	onlyOnce    sync.Once
)

// MySQL parse query and return fingeprint and placeholders.
func MySQL(q string) (string, uint32, error) {
	statement, _, err := sqlparser.Parse2(q)
	if err != nil {
		return "", 0, errors.Wrap(err, "cannot parse query")
	}

	bv := make(map[string]*query.BindVariable)
	err = sqlparser.Normalize(statement, sqlparser.NewReservedVars("", sqlparser.GetBindvars(statement)), bv)
	if err != nil {
		return "", 0, errors.Wrap(err, "cannot normalize query")
	}

	bindVars := sqlparser.GetBindvars(statement)

	return sqlparser.String(statement), uint32(len(bindVars)), nil
}

// PostgreSQL parse query and return fingeprint and placeholders.
func PostgreSQL(q string) (string, uint32, error) {
	query, err := pg_query.Normalize(q)
	if err != nil {
		return "", 0, errors.Wrap(err, "cannot normalize query")
	}

	return PostgreSQLNormalized(query)
}

// PostgreSQLNormalized parse query, which is already normalized and return fingeprint and placeholders.
func PostgreSQLNormalized(q string) (string, uint32, error) {
	// To compile regexp only once.
	onlyOnce.Do(func() {
		// PG 10 and above has $ as a placeholders.
		pgRegexp, pgRegexpErr = regexp.Compile(`[\$]{1}\d`)
	})

	if pgRegexpErr != nil {
		return "", 0, errors.Wrap(pgRegexpErr, "cannot get placeholders count")
	}

	matches := pgRegexp.FindAllString(q, -1)
	if len(matches) != 0 {
		return q, uint32(len(matches)), nil
	}

	// PG 9 has ? as a placeholders.
	return q, uint32(strings.Count(q, "?")), nil
}
