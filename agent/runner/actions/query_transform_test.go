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

package actions

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_dmlToSelect(t *testing.T) {
	t.Parallel()

	type testCase struct {
		Query     string
		Converted bool
		Expected  string
	}

	testCases := []testCase{
		{
			Query:     `SELECT nombre FROM tabla WHERE id = 0`,
			Converted: false,
			Expected:  `SELECT nombre FROM tabla WHERE id = 0`,
		},
		{
			Query:     `update ignore tabla set nombre = "carlos" where id = 0 limit 2`,
			Converted: true,
			Expected:  `SELECT nombre = "carlos" FROM tabla WHERE id = 0`,
		},
		{
			Query:     `update ignore tabla set nombre = "carlos" where id = 0`,
			Converted: true,
			Expected:  `SELECT nombre = "carlos" FROM tabla WHERE id = 0`,
		},
		{
			Query:     `update ignore tabla set nombre = "carlos" limit 1`,
			Converted: true,
			Expected:  `SELECT nombre = "carlos" FROM tabla`,
		},
		{
			Query:     `update tabla set nombre = "carlos" where id = 0 limit 2`,
			Converted: true,
			Expected:  `SELECT nombre = "carlos" FROM tabla WHERE id = 0`,
		},
		{
			Query:     `update tabla set nombre = "carlos" where id = 0`,
			Converted: true,
			Expected:  `SELECT nombre = "carlos" FROM tabla WHERE id = 0`,
		},
		{
			Query:     `update tabla set nombre = "carlos" limit 1`,
			Converted: true,
			Expected:  `SELECT nombre = "carlos" FROM tabla`,
		},
		{
			Query:     `delete from tabla`,
			Converted: true,
			Expected:  `SELECT * FROM tabla`,
		},
		{
			Query:     `delete from tabla join tabla2 on tabla.id = tabla2.tabla2_id`,
			Converted: true,
			Expected:  `SELECT 1 FROM tabla join tabla2 on tabla.id = tabla2.tabla2_id`,
		},
		{
			Query:     `insert into tabla (f1, f2, f3) values (1,2,3)`,
			Converted: true,
			Expected:  `SELECT * FROM tabla  WHERE f1=1 and f2=2 and f3=3`,
		},
		{
			Query:     `insert into tabla (f1, f2, f3) values (1,2)`,
			Converted: true,
			Expected:  `SELECT * FROM tabla  LIMIT 1`,
		},
		{
			Query:     `insert into tabla set f1="A1", f2="A2"`,
			Converted: true,
			Expected:  `SELECT * FROM tabla WHERE f1="A1" AND  f2="A2"`,
		},
		{
			Query:     "insert into `tabla-1` values(12)",
			Converted: true,
			Expected:  "SELECT * FROM `tabla-1` LIMIT 1",
		},
		{
			Query: `UPDATE
				employees2
			SET
				first_name = 'Joe',
				emp_no = 10
			WHERE
				emp_no = 3`,
			Converted: true,
			Expected:  `SELECT first_name = 'Joe',     emp_no = 10 FROM employees2 WHERE emp_no = 3`,
		},
		{
			Query: `
			/* File:movie.php Line:8 Func:update_info */
				   SELECT
				*
			FROM
				movie_info
			WHERE
				movie_id = 68357`,
			Converted: false,
			Expected:  `SELECT     *    FROM     movie_info    WHERE     movie_id = 68357`,
		},
		{
			Query: `SELECT /*+ NO_RANGE_OPTIMIZATION(t3 PRIMARY, f2_idx) */ f1
			FROM t3 WHERE f1 > 30 AND f1 < 33;`,
			Converted: false,
			Expected:  `SELECT  f1    FROM t3 WHERE f1 > 30 AND f1 < 33`,
		},
		{
			Query:     `SELECT /*+ BKA(t1) NO_BKA(t2) */ * FROM t1 INNER JOIN t2 WHERE ...;`,
			Converted: false,
			Expected:  `SELECT  * FROM t1 INNER JOIN t2 WHERE ...`,
		},
		{
			Query:     `SELECT /*+ NO_ICP(t1, t2) */ * FROM t1 INNER JOIN t2 WHERE ...;`,
			Converted: false,
			Expected:  `SELECT  * FROM t1 INNER JOIN t2 WHERE ...`,
		},
		{
			Query:     `SELECT /*+ SEMIJOIN(FIRSTMATCH, LOOSESCAN) */ * FROM t1 ...;`,
			Converted: false,
			Expected:  `SELECT  * FROM t1 ...`,
		},
		{
			Query:     `EXPLAIN SELECT /*+ NO_ICP(t1) */ * FROM t1 WHERE ...;`,
			Converted: false,
			Expected:  ``,
		},
		{
			Query:     `SELECT /*+ MERGE(dt) */ * FROM (SELECT * FROM t1) AS dt;`,
			Converted: false,
			Expected:  `SELECT  * FROM (SELECT * FROM t1) AS dt`,
		},
		{
			Query:     `INSERT /*+ SET_VAR(foreign_key_checks=OFF) */ INTO t2 VALUES(2);`,
			Converted: true,
			Expected:  `SELECT * FROM t2 LIMIT 1`,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("TestDMLToSelect %d. %s", i, tc.Query), func(t *testing.T) {
			t.Parallel()
			q, c := dmlToSelect(tc.Query)
			assert.Equal(t, tc.Converted, c)
			assert.Equal(t, tc.Expected, q)
		})
	}
}

func Test_isDMLQuery(t *testing.T) {
	assert.True(t, isDMLQuery("SELECT * FROM table"))
	assert.True(t, isDMLQuery(`update tabla set nombre = "carlos" where id = 0`))
	assert.True(t, isDMLQuery("delete from tabla join tabla2 on tabla.id = tabla2.tabla2_id"))
	assert.True(t, isDMLQuery("/*+ SET_VAR(foreign_key_checks=OFF) */ INSERT INTO t2 VALUES(2);"))
	assert.False(t, isDMLQuery("EXPLAIN SELECT * FROM table"))
}
