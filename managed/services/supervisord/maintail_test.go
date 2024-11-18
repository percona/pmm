// Copyright (C) 2023 Percona LLC
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

package supervisord

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseEvent(t *testing.T) {
	t.Parallel()
	t.Run("Normal", func(t *testing.T) {
		t.Parallel()

		log := strings.Split(`
			2019-08-08 17:09:57,284 INFO received SIGUSR2 indicating log reopen request
			2019-08-08 17:09:57,284 INFO supervisord logreopen
			2019-08-08 17:09:57,854 INFO waiting for pmm-managed to stop
			2019-08-08 17:09:59,854 INFO waiting for pmm-managed to stop
			2019-08-08 17:10:00,863 INFO stopped: pmm-managed (exit status 0)
			2019-08-08 17:10:01,932 INFO spawned: 'pmm-managed' with pid 13191
			2019-08-08 17:10:03,006 INFO success: pmm-managed entered RUNNING state, process has stayed up for > than 1 seconds (startsecs)
			2019-08-08 17:10:09,878 INFO reaped unknown pid 12411
			2019-08-08 17:10:27,686 INFO spawned: 'dashboard-upgrade' with pid 13888
			2019-08-08 17:10:27,686 INFO success: dashboard-upgrade entered RUNNING state, process has stayed up for > than 0 seconds (startsecs)
			2019-08-08 17:10:27,761 INFO exited: dashboard-upgrade (exit status 0; expected)
		`, "\n")

		var actual []*event
		for _, line := range log {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if e := parseEvent(line); e != nil {
				actual = append(actual, e)
			}
		}
		expected := []*event{
			{Time: time.Date(2019, 8, 8, 17, 9, 57, 284000000, time.UTC), Type: logReopen, Program: "supervisord"},
			{Time: time.Date(2019, 8, 8, 17, 9, 57, 854000000, time.UTC), Type: stopping, Program: "pmm-managed"},
			{Time: time.Date(2019, 8, 8, 17, 9, 59, 854000000, time.UTC), Type: stopping, Program: "pmm-managed"},
			{Time: time.Date(2019, 8, 8, 17, 10, 0, 863000000, time.UTC), Type: stopped, Program: "pmm-managed"},
			{Time: time.Date(2019, 8, 8, 17, 10, 1, 932000000, time.UTC), Type: starting, Program: "pmm-managed"},
			{Time: time.Date(2019, 8, 8, 17, 10, 3, 6000000, time.UTC), Type: running, Program: "pmm-managed"},
			{Time: time.Date(2019, 8, 8, 17, 10, 27, 686000000, time.UTC), Type: starting, Program: "dashboard-upgrade"},
			{Time: time.Date(2019, 8, 8, 17, 10, 27, 686000000, time.UTC), Type: running, Program: "dashboard-upgrade"},
			{Time: time.Date(2019, 8, 8, 17, 10, 27, 761000000, time.UTC), Type: exitedExpected, Program: "dashboard-upgrade"},
		}
		assert.Equal(t, expected, actual)
	})

	t.Run("Fatal", func(t *testing.T) {
		t.Parallel()

		log := strings.Split(`
			2019-08-09 09:18:25,667 INFO spawned: 'pmm-update' with pid 11410
			2019-08-09 09:18:26,539 INFO exited: pmm-update (exit status 0; not expected)
			2019-08-09 09:18:27,543 INFO spawned: 'pmm-update' with pid 11421
			2019-08-09 09:18:28,324 INFO exited: pmm-update (exit status 0; not expected)
			2019-08-09 09:18:30,335 INFO spawned: 'pmm-update' with pid 11432
			2019-08-09 09:18:31,109 INFO exited: pmm-update (exit status 0; not expected)
			2019-08-09 09:18:34,119 INFO spawned: 'pmm-update' with pid 11443
			2019-08-09 09:18:34,883 INFO exited: pmm-update (exit status 0; not expected)
			2019-08-09 09:18:35,885 INFO gave up: pmm-update entered FATAL state, too many start retries too quickly
		`, "\n")

		var actual []*event
		for _, line := range log {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if e := parseEvent(line); e != nil {
				actual = append(actual, e)
			}
		}
		expected := []*event{
			{Time: time.Date(2019, 8, 9, 9, 18, 25, 667000000, time.UTC), Type: starting, Program: "pmm-update"},
			{Time: time.Date(2019, 8, 9, 9, 18, 26, 539000000, time.UTC), Type: exitedUnexpected, Program: "pmm-update"},
			{Time: time.Date(2019, 8, 9, 9, 18, 27, 543000000, time.UTC), Type: starting, Program: "pmm-update"},
			{Time: time.Date(2019, 8, 9, 9, 18, 28, 324000000, time.UTC), Type: exitedUnexpected, Program: "pmm-update"},
			{Time: time.Date(2019, 8, 9, 9, 18, 30, 335000000, time.UTC), Type: starting, Program: "pmm-update"},
			{Time: time.Date(2019, 8, 9, 9, 18, 31, 109000000, time.UTC), Type: exitedUnexpected, Program: "pmm-update"},
			{Time: time.Date(2019, 8, 9, 9, 18, 34, 119000000, time.UTC), Type: starting, Program: "pmm-update"},
			{Time: time.Date(2019, 8, 9, 9, 18, 34, 883000000, time.UTC), Type: exitedUnexpected, Program: "pmm-update"},
			{Time: time.Date(2019, 8, 9, 9, 18, 35, 885000000, time.UTC), Type: fatal, Program: "pmm-update"},
		}
		assert.Equal(t, expected, actual)
	})
}
