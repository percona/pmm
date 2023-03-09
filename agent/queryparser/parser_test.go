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
	"testing"

	"github.com/stretchr/testify/require"
)

type testCase struct {
	Name     string
	Query    string
	Comments []string
}

func TestMySQLComments(t *testing.T) {
	testCases := []testCase{
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
			Query: `SELECT * FROM people /*!
				special multiline comment case 
				with new line
				 */`,
			Comments: []string{"special multiline comment case with new line"},
		},
		{
			Name: "Second special multiline comment case with new line",
			Query: `SELECT * FROM people /*+ second special 
				  multiline comment case 
				with new line */`,
			Comments: []string{"second special multiline comment case with new line"},
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
