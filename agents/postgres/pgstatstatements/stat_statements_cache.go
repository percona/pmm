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

package pgstatstatements

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-agent/agents/postgres/parser"
	"github.com/percona/pmm-agent/utils/truncate"
)

// statStatementCacheStats contains statStatementCache statistics.
type statStatementCacheStats struct {
	current  uint
	updatedN uint
	addedN   uint
	removedN uint
	oldest   time.Time
	newest   time.Time
}

func (s statStatementCacheStats) String() string {
	d := s.newest.Sub(s.oldest)
	return fmt.Sprintf("current=%d: updated=%d added=%d removed=%d; %s - %s (%s)",
		s.current, s.updatedN, s.addedN, s.removedN,
		s.oldest.UTC().Format("2006-01-02T15:04:05Z"), s.newest.UTC().Format("2006-01-02T15:04:05Z"), d)
}

// statStatementCache provides cached access to pg_stat_statements.
// It retains data longer than this table.
type statStatementCache struct {
	retain time.Duration
	l      *logrus.Entry

	rw       sync.RWMutex
	items    map[int64]*pgStatStatementsExtended
	added    map[int64]time.Time
	updatedN uint
	addedN   uint
	removedN uint
}

// newStatStatementCache creates new statStatementCache.
func newStatStatementCache(retain time.Duration, l *logrus.Entry) *statStatementCache {
	return &statStatementCache{
		retain: retain,
		l:      l,
		items:  make(map[int64]*pgStatStatementsExtended),
		added:  make(map[int64]time.Time),
	}
}

// getStatStatementsExtended returns the current state of pg_stat_statements table with extended information (database, username, tables)
// and the previous cashed state.
func (ssc *statStatementCache) getStatStatementsExtended(ctx context.Context, q *reform.Querier) (current, prev map[int64]*pgStatStatementsExtended, err error) {
	var totalN, newN, newSharedN, oldN int
	start := time.Now()
	defer func() {
		dur := time.Since(start)
		ssc.l.Debugf("Selected %d rows from pg_stat_statements in %s: %d new (%d shared tables), %d old.", totalN, dur, newN, newSharedN, oldN)
	}()

	ssc.rw.RLock()
	current = make(map[int64]*pgStatStatementsExtended, len(ssc.items))
	prev = make(map[int64]*pgStatStatementsExtended, len(ssc.items))
	for k, v := range ssc.items {
		prev[k] = v
	}
	ssc.rw.RUnlock()

	// load all databases and usernames first as we can't use querier while iterating over rows below
	databases := queryDatabases(q)
	usernames := queryUsernames(q)

	// the same query can appear several times (with different database and/or username),
	// so cache results of the current iteration too
	tables := make(map[int64][]string)

	rows, err := q.SelectRows(pgStatStatementsView, "WHERE queryid IS NOT NULL AND query IS NOT NULL")
	if err != nil {
		err = errors.Wrap(err, "failed to query pg_stat_statements")
		return
	}
	defer rows.Close() //nolint:errcheck

	for ctx.Err() == nil {
		var row pgStatStatements
		if err = q.NextRow(&row, rows); err != nil {
			if err == reform.ErrNoRows {
				err = nil
			}
			break
		}
		totalN++

		c := &pgStatStatementsExtended{
			pgStatStatements: row,
			Database:         databases[row.DBID],
			Username:         usernames[row.UserID],
		}

		if p := prev[c.QueryID]; p != nil {
			oldN++

			// use previous values
			c.Tables = p.Tables
			c.Query, c.IsQueryTruncated = p.Query, p.IsQueryTruncated
		} else {
			newN++

			// do not extract tables again if we saw this query during this iteration already
			if tables[c.QueryID] == nil {
				tables[c.QueryID] = extractTables(c.Query, ssc.l)
			} else {
				newSharedN++
			}

			c.Tables = tables[c.QueryID]
			c.Query, c.IsQueryTruncated = truncate.Query(c.Query)
		}

		current[c.QueryID] = c
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}

	if err != nil {
		err = errors.Wrap(err, "failed to fetch pg_stat_statements")
	}
	return
}

// stats returns statStatementCache statistics.
func (ssc *statStatementCache) stats() statStatementCacheStats {
	ssc.rw.RLock()
	defer ssc.rw.RUnlock()

	oldest := time.Now().Add(retainStatStatements)
	var newest time.Time
	for _, t := range ssc.added {
		if oldest.After(t) {
			oldest = t
		}
		if newest.Before(t) {
			newest = t
		}
	}

	return statStatementCacheStats{
		current:  uint(len(ssc.added)),
		updatedN: ssc.updatedN,
		addedN:   ssc.addedN,
		removedN: ssc.removedN,
		oldest:   oldest,
		newest:   newest,
	}
}

// refresh removes expired items in cache, then adds current items.
func (ssc *statStatementCache) refresh(current map[int64]*pgStatStatementsExtended) {
	ssc.rw.Lock()
	defer ssc.rw.Unlock()

	now := time.Now()

	for k, t := range ssc.added {
		if now.Sub(t) > ssc.retain {
			ssc.removedN++
			delete(ssc.items, k)
			delete(ssc.added, k)
		}
	}

	for k, v := range current {
		if _, ok := ssc.items[k]; ok {
			ssc.updatedN++
		} else {
			ssc.addedN++
		}
		ssc.items[k] = v
		ssc.added[k] = now
	}
}

func queryDatabases(q *reform.Querier) map[int64]string {
	structs, err := q.SelectAllFrom(pgStatDatabaseView, "")
	if err != nil {
		return nil
	}

	res := make(map[int64]string, len(structs))
	for _, str := range structs {
		d := str.(*pgStatDatabase)
		res[d.DatID] = pointer.GetString(d.DatName)
	}
	return res
}

func queryUsernames(q *reform.Querier) map[int64]string {
	structs, err := q.SelectAllFrom(pgUserView, "")
	if err != nil {
		return nil
	}

	res := make(map[int64]string, len(structs))
	for _, str := range structs {
		u := str.(*pgUser)
		res[u.UserID] = pointer.GetString(u.UserName)
	}
	return res
}

func extractTables(query string, l *logrus.Entry) []string {
	start := time.Now()
	t, _ := truncate.Query(query)
	tables, err := parser.ExtractTables(query)
	if err != nil {
		l.Warnf("Can't extract table names from query %s: %s", t, err)
		return []string{} // not-nil to cache for the current iteration
	}

	dur := time.Since(start)
	logf := l.Debugf
	if dur > 500*time.Millisecond {
		logf = l.Warnf
	}
	logf("Extracted table names %v from query %s. It took %s.", tables, t, dur)
	return tables
}
