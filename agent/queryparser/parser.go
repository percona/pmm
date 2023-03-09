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

// Package queryparser provides functionality for queries parsing.
package queryparser

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"vitess.io/vitess/go/vt/proto/query"
	"vitess.io/vitess/go/vt/sqlparser"
)

// MySQL parse query and return fingeprint and placeholders.
func MySQL(q string) (string, uint32, error) {
	normalizedQuery, _, err := sqlparser.Parse2(q)
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

// MySQLComments parse query and return its comments. Can parse multi comments.
// Multi comments support should be dropped in future MySQL versions.
// Doc: https://dev.mysql.com/doc/refman/8.0/en/comments.html
func MySQLComments(q string) ([]string, error) {
	// sqlparser.ExtractMysqlComment(q) doesnt work properly
	// input: SELECT * FROM people /*! bla bla */ WHERE name = 'john'
	// output: ECT * FROM people /*! bla bla */ WHERE name = 'joh
	r, err := regexp.Compile("(?s)\\/\\*(.*?) \\*\\/")
	if err != nil {
		return nil, err
	}

	// comments using comment as a key to avoid duplicates
	comments := make(map[string]bool)
	for _, v := range r.FindAllStringSubmatch(strings.Replace(q, "\n", "", -1), -1) {
		if len(v) < 2 {
			continue
		}

		// replace all mutations of multiline comment
		// /*! and /*+
		replacer := strings.NewReplacer("!", "", "+", "")
		comments[replacer.Replace(v[1])[1:]] = true
	}

	hashIndex := strings.Index(q, "#")
	if hashIndex > -1 {
		comments[q[(hashIndex+2):]] = true
	}

	dashIndex := strings.Index(q, "--")
	if dashIndex > -1 {
		comments[q[(dashIndex+3):]] = true
	}

	var res []string
	for k, _ := range comments {
		res = append(res, k)
	}

	return res, nil
}
