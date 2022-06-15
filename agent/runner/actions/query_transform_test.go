// pmm-agent
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

package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDMLToSelect(t *testing.T) {
	q := dmlToSelect(`update ignore tabla set nombre = "carlos" where id = 0 limit 2`)
	assert.Equal(t, `SELECT nombre = "carlos" FROM tabla WHERE id = 0`, q)

	q = dmlToSelect(`update ignore tabla set nombre = "carlos" where id = 0`)
	assert.Equal(t, `SELECT nombre = "carlos" FROM tabla WHERE id = 0`, q)

	q = dmlToSelect(`update ignore tabla set nombre = "carlos" limit 1`)
	assert.Equal(t, `SELECT nombre = "carlos" FROM tabla`, q)

	q = dmlToSelect(`update tabla set nombre = "carlos" where id = 0 limit 2`)
	assert.Equal(t, `SELECT nombre = "carlos" FROM tabla WHERE id = 0`, q)

	q = dmlToSelect(`update tabla set nombre = "carlos" where id = 0`)
	assert.Equal(t, `SELECT nombre = "carlos" FROM tabla WHERE id = 0`, q)

	q = dmlToSelect(`update tabla set nombre = "carlos" limit 1`)
	assert.Equal(t, `SELECT nombre = "carlos" FROM tabla`, q)

	q = dmlToSelect(`delete from tabla`)
	assert.Equal(t, `SELECT * FROM tabla`, q)

	q = dmlToSelect(`delete from tabla join tabla2 on tabla.id = tabla2.tabla2_id`)
	assert.Equal(t, `SELECT 1 FROM tabla join tabla2 on tabla.id = tabla2.tabla2_id`, q)

	q = dmlToSelect(`insert into tabla (f1, f2, f3) values (1,2,3)`)
	assert.Equal(t, `SELECT * FROM tabla  WHERE f1=1 and f2=2 and f3=3`, q)

	q = dmlToSelect(`insert into tabla (f1, f2, f3) values (1,2)`)
	assert.Equal(t, `SELECT * FROM tabla  LIMIT 1`, q)

	q = dmlToSelect(`insert into tabla set f1="A1", f2="A2"`)
	assert.Equal(t, `SELECT * FROM tabla WHERE f1="A1" AND  f2="A2"`, q)

	q = dmlToSelect(`replace into tabla set f1="A1", f2="A2"`)
	assert.Equal(t, `SELECT * FROM tabla WHERE f1="A1" AND  f2="A2"`, q)

	q = dmlToSelect("insert into `tabla-1` values(12)")
	assert.Equal(t, "SELECT * FROM `tabla-1` LIMIT 1", q)

	q = dmlToSelect(`UPDATE
  employees2
SET
  first_name = 'Joe',
  emp_no = 10
WHERE
  emp_no = 3`)
	assert.Equal(t, "SELECT first_name = 'Joe',   emp_no = 10 FROM employees2 WHERE emp_no = 3", q)

	q = dmlToSelect(`UPDATE employees2 SET first_name = 'Joe', emp_no = 10 WHERE emp_no = 3`)
	assert.Equal(t, "SELECT first_name = 'Joe', emp_no = 10 FROM employees2 WHERE emp_no = 3", q)
}
