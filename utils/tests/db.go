// pmm-agent
// Copyright (C) 2018 Percona LLC
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

package tests

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// waitForFixtures waits up to 30 seconds to database fixtures (test_db) to be loaded.
func waitForFixtures(tb testing.TB, db *sql.DB) {
	var id int
	var err error
	for i := 0; i < 30; i++ {
		if err = db.QueryRow("SELECT /* pmm-agent-tests:waitForFixtures */ id FROM city LIMIT 1").Scan(&id); err == nil {
			return
		}
		time.Sleep(time.Second)
	}
	require.NoError(tb, err)
}
