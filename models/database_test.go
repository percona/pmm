// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package models_test

import (
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-managed/utils/tests"
)

func assertCantBeNull(t *testing.T, err error, column string) {
	t.Helper()

	require.IsType(t, &mysql.MySQLError{}, err)
	mysqlErr := err.(*mysql.MySQLError)
	assert.EqualValues(t, 1048, mysqlErr.Number)
	assert.Equal(t, fmt.Sprintf("Column '%s' cannot be null", column), mysqlErr.Message)
}

func assertDuplicate(t *testing.T, err error, value, key string) {
	t.Helper()

	require.IsType(t, &mysql.MySQLError{}, err)
	mysqlErr := err.(*mysql.MySQLError)
	assert.EqualValues(t, 1062, mysqlErr.Number)
	assert.Equal(t, fmt.Sprintf("Duplicate entry '%s' for key '%s'", value, key), mysqlErr.Message)
}

func TestDatabaseUniqueIndexes(t *testing.T) {
	db := tests.OpenTestDB(t)
	defer func() {
		require.NoError(t, db.Close())
	}()

	var err error

	t.Run("Nodes", func(t *testing.T) {
		_, err = db.Exec("INSERT INTO nodes (node_id) VALUES (NULL)")
		assertCantBeNull(t, err, "node_id")
		_, err = db.Exec("INSERT INTO nodes (node_type) VALUES (NULL)")
		assertCantBeNull(t, err, "node_type")
		_, err = db.Exec("INSERT INTO nodes (node_name) VALUES (NULL)")
		assertCantBeNull(t, err, "node_name")

		_, err = db.Exec("INSERT INTO nodes (node_id, node_type, node_name) VALUES ('1', '', 'name')")
		assert.NoError(t, err)
		_, err = db.Exec("INSERT INTO nodes (node_id, node_type, node_name) VALUES ('1', '', 'other name')")
		assertDuplicate(t, err, "1", "PRIMARY")

		_, err = db.Exec("INSERT INTO nodes (node_id, node_type, node_name) VALUES ('2', '', 'name')")
		assertDuplicate(t, err, "name", "node_name")

		_, err = db.Exec("INSERT INTO nodes (node_id, node_type, node_name, machine_id) VALUES ('31', '', 'name31', NULL)")
		assert.NoError(t, err)
		_, err = db.Exec("INSERT INTO nodes (node_id, node_type, node_name, machine_id) VALUES ('32', '', 'name32', 'machine-id')")
		assert.NoError(t, err)
		_, err = db.Exec("INSERT INTO nodes (node_id, node_type, node_name, machine_id) VALUES ('33', '', 'name33', 'machine-id')")
		assertDuplicate(t, err, "machine-id", "machine_id")
	})
}
