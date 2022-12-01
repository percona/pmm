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
	"testing"

	"github.com/stretchr/testify/assert"
)

type test struct {
	Query                     string
	ExpectedQuery             string
	ExpectedPlaceHoldersCount uint32
}

func TestMySQL(t *testing.T) {
	sqls := []test{
		{
			Query:                     "SELECT name FROM people where city = 'Paris'",
			ExpectedQuery:             "select `name` from people where city = :1",
			ExpectedPlaceHoldersCount: 1,
		},
		{
			Query:                     "SELECT name FROM people where city = ?",
			ExpectedQuery:             "select `name` from people where city = :v1",
			ExpectedPlaceHoldersCount: 1,
		},
		{
			Query:                     "INSERT INTO people VALUES('John', 'Paris', 70010)",
			ExpectedQuery:             "insert into people values (:1, :2, :3)",
			ExpectedPlaceHoldersCount: 3,
		},
		{
			Query:                     "INSERT INTO people VALUES(?, ?, ?)",
			ExpectedQuery:             "insert into people values (:v1, :v2, :v3)",
			ExpectedPlaceHoldersCount: 3,
		},
	}

	for _, sql := range sqls {
		query, placeholdersCount, err := MySQL(sql.Query)
		assert.NoError(t, err)
		assert.Equal(t, sql.ExpectedQuery, query)
		assert.Equal(t, sql.ExpectedPlaceHoldersCount, placeholdersCount)
	}
}

func TestPostgreSQL(t *testing.T) {
	sqls := []test{
		{
			Query:                     "SELECT name FROM people where city = 'Paris'",
			ExpectedQuery:             "SELECT name FROM people where city = $1",
			ExpectedPlaceHoldersCount: 1,
		},
		{
			Query:                     "SELECT name FROM people where city = ?",
			ExpectedQuery:             "SELECT name FROM people where city = ?",
			ExpectedPlaceHoldersCount: 1,
		},
		{
			Query:                     "INSERT INTO people VALUES('John', 'Paris', 70010)",
			ExpectedQuery:             "INSERT INTO people VALUES($1, $2, $3)",
			ExpectedPlaceHoldersCount: 3,
		},
		{
			Query:                     "INSERT INTO people VALUES(?, ?, ?)",
			ExpectedQuery:             "INSERT INTO people VALUES(?, ?, ?)",
			ExpectedPlaceHoldersCount: 3,
		},
	}

	for _, sql := range sqls {
		query, placeholdersCount, err := PostgreSQL(sql.Query)
		assert.NoError(t, err)
		assert.Equal(t, sql.ExpectedQuery, query)
		assert.Equal(t, sql.ExpectedPlaceHoldersCount, placeholdersCount)
	}
}
