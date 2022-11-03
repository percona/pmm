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
	"github.com/pkg/errors"
	"vitess.io/vitess/go/vt/proto/query"
	"vitess.io/vitess/go/vt/sqlparser"
)

// MySQL parse query and return fingeprint and placeholders.
func MySQL(example string) (string, uint32, error) {
	normalizedQuery, _, err := sqlparser.Parse2(example)
	if err != nil {
		return "", 0, errors.Wrap(err, "cannot parse query")
	}

	bv := make(map[string]*query.BindVariable)
	err = sqlparser.Normalize(normalizedQuery, sqlparser.NewReservedVars("", sqlparser.GetBindvars(normalizedQuery)), bv)
	if err != nil {
		return "", 0, errors.Wrap(err, "cannot normalize query")
	}

	parsedQuery := sqlparser.NewParsedQuery(normalizedQuery)
	bindVars := sqlparser.GetBindvars(normalizedQuery)

	return parsedQuery.Query, uint32(len(bindVars)), nil
}
