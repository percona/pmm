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
	"github.com/stretchr/testify/require"
)

type testCase struct {
	Query                     string
	ExpectedQuery             string
	ExpectedPlaceHoldersCount uint32
}

func TestMySQL(t *testing.T) {
	sqls := []testCase{
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
		{
			Query: `SELECT t.table_schema, t.table_name, column_name, auto_increment, pow(2, case data_type when
				'tinyint' then 7 when 'smallint' then 15 when 'mediumint' then 23 when 'int' then 31 when 'bigint' then 63 
				end +(column_type like '% unsigned')) -1 as max_int FROM information_schema.columns c STRAIGHT_JOIN 
				information_schema.tables t ON BINARY t.table_schema = c.table_schema AND BINARY t.table_name = c.table_name
		  		WHERE c.extra = 'auto_increment' AND t.auto_increment IS NOT NULL`,
			ExpectedQuery: "select t.table_schema, t.table_name, column_name, `auto_increment`, pow(:1, case " +
				"data_type when :2 then :3 when :4 then :5 when :6 then :7 when :8 then :9 when :10 then :11 end + " +
				"(column_type like :12)) - :13 as max_int from information_schema.`columns` as c straight_join information_schema.`tables` " +
				"as t on convert(t.table_schema, BINARY) = c.table_schema and convert(t.table_name, BINARY) = c.table_name where c.extra = :14 " +
				"and t.`auto_increment` is not null",
			ExpectedPlaceHoldersCount: 14,
		},
	}

	for _, sql := range sqls {
		query, placeholdersCount, err := MySQL(sql.Query)
		require.NoError(t, err)
		assert.Equal(t, sql.ExpectedQuery, query)
		assert.Equal(t, sql.ExpectedPlaceHoldersCount, placeholdersCount)
	}
}

type testCaseComments struct {
	Name     string
	Query    string
	Comments []string
}

func TestMySQLComments(t *testing.T) {
	testCases := []testCaseComments{
		{
			Name: "No comment",
			Query: `SELECT * FROM people WHERE name = 'John'
				 AND name != 'Doe'`,
			Comments: nil,
		},
		{
			Name:     "Dash comment",
			Query:    `SELECT * FROM people -- dash comment`,
			Comments: []string{"dash comment"},
		},
		{
			Name: "Hash comment",
			Query: `SELECT * FROM people # hash comment
			WHERE name = 'John'
			`,
			Comments: []string{"hash comment"},
		},
		{
			Name:     "Multiline comment",
			Query:    `SELECT * FROM people /* multiline comment */`,
			Comments: []string{"multiline comment"},
		},
		{
			Name: "Multiline comment with new line",
			Query: `SELECT * FROM people /* multiline comment 
				with new line */`,
			Comments: []string{"multiline comment with new line"},
		},
		{
			Name: "Special multiline comment case with new line",
			Query: `SELECT * FROM people /*!80000 
				special multiline comment case 
				with new line
				 */`,
			Comments: []string{"!80000 special multiline comment case with new line"},
		},
		{
			Name: "Second special multiline comment case with new line",
			Query: `SELECT * FROM people /*+ BKA(t1) 
				second special multiline comment case 
				with new line */`,
			Comments: []string{"+ BKA(t1) second special multiline comment case with new line"},
		},
		{
			Name: "Multicomment case with new line",
			Query: `SELECT * FROM people /*
				multicomment case 
				  with new line 
				 */ WHERE name = 'John' # John
				 AND name != 'Doe'`,
			Comments: []string{"multicomment case with new line", "John"},
		},
	}

	for _, c := range testCases {
		t.Run(c.Name, func(t *testing.T) {
			comments, err := MySQLComments(c.Query)
			require.NoError(t, err)
			require.Equal(t, c.Comments, comments)
		})
	}
}

func TestPostgreSQLComments(t *testing.T) {
	testCases := []testCaseComments{
		{
			Name: "No comment",
			Query: `SELECT * FROM people WHERE name = 'John'
				 AND name != 'Doe'`,
			Comments: nil,
		},
		{
			Name:     "Dash comment",
			Query:    `SELECT * FROM people -- dash comment`,
			Comments: []string{"dash comment"},
		},
		{
			Name: "Multiline comment case with new line",
			Query: `SELECT * FROM people /* new
				* multiline comment case 
				* with new line
				 */`,
			Comments: []string{"new multiline comment case with new line"},
		},
		{
			Name: "Second multiline comment case with new line",
			Query: `SELECT * FROM people /* test 
				with new line */`,
			Comments: []string{"test with new line"},
		},
	}

	for _, c := range testCases {
		t.Run(c.Name, func(t *testing.T) {
			comments, err := PostgreSQLComments(c.Query)
			require.NoError(t, err)
			require.Equal(t, c.Comments, comments)
		})
	}
}
