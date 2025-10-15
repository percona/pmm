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

package pgstatmonitor

import "fmt"

//go:generate ../../../../bin/reform

// pgStatDatabase represents a row in pg_stat_database view.
//
//reform:pg_catalog.pg_stat_database
type pgStatDatabase struct { //nolint:recvcheck
	DatID   int64   `reform:"datid"`
	DatName *string `reform:"datname"`
}

// pgUser represents a row in pg_user view.
//
//reform:pg_catalog.pg_user
type pgUser struct { //nolint:recvcheck
	UserID   int64   `reform:"usesysid"`
	UserName *string `reform:"usename"`
}

// pgStatMonitorSettings represents a row in pg_stat_monitor_settings view before 1.0.0-rc.2.
//
//reform:pg_stat_monitor_settings
type pgStatMonitorSettings struct { //nolint:recvcheck
	Name  string `reform:"name"`
	Value int64  `reform:"value"`
}

// pgStatMonitorSettingsTextValue represents a row in pg_stat_monitor_settings view from 1.0.0-rc.2 until 2.0.0-dev (2.0.0.-dev excluded).
//
//reform:pg_stat_monitor_settings
type pgStatMonitorSettingsTextValue struct { //nolint:recvcheck
	Name  string `reform:"name"`
	Value string `reform:"value"`
}

// pgStatMonitorExtended contains pgStatMonitor data and extends it with database, username and tables data.
// It's made for performance reason.
type pgStatMonitorExtended struct {
	pgStatMonitor

	Fingerprint      string
	Example          string
	Database         string
	Username         string
	IsQueryTruncated bool
}

func (e *pgStatMonitorExtended) String() string {
	return fmt.Sprintf("%q %q %v: %s: %s (truncated = %t)",
		e.Database, e.Username, e.Relations, e.QueryID, e.Query, e.IsQueryTruncated)
}
