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

// Package tests provides testing until functions.
package tests

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// waitForTestDataLoad waits up to 30 seconds to test dataset to be loaded.
func waitForTestDataLoad(tb testing.TB, db *sql.DB) {
	tb.Helper()

	var count int
	var err error
	for i := 0; i < 30; i++ {
		if err = db.QueryRow("SELECT /* pmm-agent-tests:waitForTestDataLoad */ COUNT(*) FROM city").Scan(&count); err == nil {
			return
		}

		// Size on test dataset https://github.com/AlekSi/test_db/blob/4c673cc28648568fc23d35e86f280f411498620e/mysql/world/world.sql#L4125
		if count == 4079 {
			return
		}
		time.Sleep(time.Second)
	}
	require.NoError(tb, err)
}
