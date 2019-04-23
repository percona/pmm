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

package perfschema

//go:generate reform

// eventsStatementsSummaryByDigest represents a row in performance_schema.events_statements_summary_by_digest table.
//reform:performance_schema.events_statements_summary_by_digest
type eventsStatementsSummaryByDigest struct {
	SchemaName              *string `reform:"SCHEMA_NAME"`
	Digest                  *string `reform:"DIGEST"`      // MD5 of DigestText
	DigestText              *string `reform:"DIGEST_TEXT"` // query without values
	CountStar               int64   `reform:"COUNT_STAR"`
	SumTimerWait            int64   `reform:"SUM_TIMER_WAIT"`
	MinTimerWait            int64   `reform:"MIN_TIMER_WAIT"`
	AvgTimerWait            int64   `reform:"AVG_TIMER_WAIT"`
	MaxTimerWait            int64   `reform:"MAX_TIMER_WAIT"`
	SumLockTime             int64   `reform:"SUM_LOCK_TIME"`
	SumErrors               int64   `reform:"SUM_ERRORS"`
	SumWarnings             int64   `reform:"SUM_WARNINGS"`
	SumRowsAffected         int64   `reform:"SUM_ROWS_AFFECTED"`
	SumRowsSent             int64   `reform:"SUM_ROWS_SENT"`
	SumRowsExamined         int64   `reform:"SUM_ROWS_EXAMINED"`
	SumCreatedTmpDiskTables int64   `reform:"SUM_CREATED_TMP_DISK_TABLES"`
	SumCreatedTmpTables     int64   `reform:"SUM_CREATED_TMP_TABLES"`
	SumSelectFullJoin       int64   `reform:"SUM_SELECT_FULL_JOIN"`
	SumSelectFullRangeJoin  int64   `reform:"SUM_SELECT_FULL_RANGE_JOIN"`
	SumSelectRange          int64   `reform:"SUM_SELECT_RANGE"`
	SumSelectRangeCheck     int64   `reform:"SUM_SELECT_RANGE_CHECK"`
	SumSelectScan           int64   `reform:"SUM_SELECT_SCAN"`
	SumSortMergePasses      int64   `reform:"SUM_SORT_MERGE_PASSES"`
	SumSortRange            int64   `reform:"SUM_SORT_RANGE"`
	SumSortRows             int64   `reform:"SUM_SORT_ROWS"`
	SumSortScan             int64   `reform:"SUM_SORT_SCAN"`
	SumNoIndexUsed          int64   `reform:"SUM_NO_INDEX_USED"`
	SumNoGoodIndexUsed      int64   `reform:"SUM_NO_GOOD_INDEX_USED"`
	// FirstSeen               time.Time `reform:"FIRST_SEEN"`
	// LastSeen                time.Time `reform:"LAST_SEEN"`
}

// eventsStatementsHistory represents a row in performance_schema.events_statements_history table.
//reform:performance_schema.events_statements_history
type eventsStatementsHistory struct {
	// ThreadID   int64   `reform:"THREAD_ID"`
	// EventID    int64   `reform:"EVENT_ID"`
	// EndEventID *int64  `reform:"END_EVENT_ID"`
	// EventName  string  `reform:"EVENT_NAME"`
	// Source     *string `reform:"SOURCE"`
	// TimerStart *int64  `reform:"TIMER_START"`
	// TimerEnd   *int64  `reform:"TIMER_END"`
	// TimerWait  *int64  `reform:"TIMER_WAIT"`
	// LockTime   int64   `reform:"LOCK_TIME"`
	SQLText *string `reform:"SQL_TEXT"`
	Digest  *string `reform:"DIGEST"`
	// DigestText    *string `reform:"DIGEST_TEXT"`
	CurrentSchema *string `reform:"CURRENT_SCHEMA"`
	// ObjectType           *string `reform:"OBJECT_TYPE"`
	// ObjectSchema         *string `reform:"OBJECT_SCHEMA"`
	// ObjectName           *string `reform:"OBJECT_NAME"`
	// ObjectInstanceBegin  *int64  `reform:"OBJECT_INSTANCE_BEGIN"`
	// MySQLErrno           *int32  `reform:"MYSQL_ERRNO"`
	// ReturnedSqlstate     *string `reform:"RETURNED_SQLSTATE"`
	// MessageText          *string `reform:"MESSAGE_TEXT"`
	// Errors               int64   `reform:"ERRORS"`
	// Warnings             int64   `reform:"WARNINGS"`
	// RowsAffected         int64   `reform:"ROWS_AFFECTED"`
	// RowsSent             int64   `reform:"ROWS_SENT"`
	// RowsExamined         int64   `reform:"ROWS_EXAMINED"`
	// CreatedTmpDiskTables int64   `reform:"CREATED_TMP_DISK_TABLES"`
	// CreatedTmpTables     int64   `reform:"CREATED_TMP_TABLES"`
	// SelectFullJoin       int64   `reform:"SELECT_FULL_JOIN"`
	// SelectFullRangeJoin  int64   `reform:"SELECT_FULL_RANGE_JOIN"`
	// SelectRange          int64   `reform:"SELECT_RANGE"`
	// SelectRangeCheck     int64   `reform:"SELECT_RANGE_CHECK"`
	// SelectScan           int64   `reform:"SELECT_SCAN"`
	// SortMergePasses      int64   `reform:"SORT_MERGE_PASSES"`
	// SortRange            int64   `reform:"SORT_RANGE"`
	// SortRows             int64   `reform:"SORT_ROWS"`
	// SortScan             int64   `reform:"SORT_SCAN"`
	// MoIndexUsed          int64   `reform:"NO_INDEX_USED"`
	// MoGoodIndexUsed      int64   `reform:"NO_GOOD_INDEX_USED"`
}

// setupConsumers represents a row in performance_schema.setup_consumers table.
//reform:performance_schema.setup_consumers
type setupConsumers struct {
	Name    string `reform:"NAME"`
	Enabled string `reform:"ENABLED"`
}

// setupInstruments represents a row in performance_schema.setup_instruments table.
//reform:performance_schema.setup_instruments
type setupInstruments struct {
	Name    string  `reform:"NAME"`
	Enabled string  `reform:"ENABLED"`
	Timed   *string `reform:"TIMED"` // nullable in 8.0
}
