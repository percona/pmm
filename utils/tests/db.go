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
