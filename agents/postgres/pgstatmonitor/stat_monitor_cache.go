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

package pgstatmonitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/percona/pmm-agent/utils/truncate"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/go-mysql/query"
)

// statMonitorCacheStats contains statMonitorCache statistics.
type statMonitorCacheStats struct {
	current  uint
	updatedN uint
	addedN   uint
	removedN uint
	oldest   time.Time
	newest   time.Time
}

func (s statMonitorCacheStats) String() string {
	d := s.newest.Sub(s.oldest)
	return fmt.Sprintf("current=%d: updated=%d added=%d removed=%d; %s - %s (%s)",
		s.current, s.updatedN, s.addedN, s.removedN,
		s.oldest.UTC().Format("2006-01-02T15:04:05Z"), s.newest.UTC().Format("2006-01-02T15:04:05Z"), d)
}

// statMonitorCache provides cached access to pg_stat_monitor.
// It retains data longer than this table.
type statMonitorCache struct {
	retain time.Duration
	l      *logrus.Entry

	rw       sync.RWMutex
	items    map[string]*pgStatMonitorExtended
	added    map[string]time.Time
	updatedN uint
	addedN   uint
	removedN uint
}

// newStatMonitorCache creates new statMonitorCache.
func newStatMonitorCache(retain time.Duration, l *logrus.Entry) *statMonitorCache {
	return &statMonitorCache{
		retain: retain,
		l:      l,
		items:  make(map[string]*pgStatMonitorExtended),
		added:  make(map[string]time.Time),
	}
}

// getStatMonitorExtended returns the current state of pg_stat_monitor table with extended information (database, username)
// and the previous cashed state.
func (ssc *statMonitorCache) getStatMonitorExtended(ctx context.Context, q *reform.Querier, normalizedQuery, disableQueryExamples bool) (current, prev map[string]*pgStatMonitorExtended, err error) {
	var totalN, newN, newSharedN, oldN int
	start := time.Now()
	defer func() {
		dur := time.Since(start)
		ssc.l.Debugf("Selected %d rows from pg_stat_monitor in %s: %d new (%d shared tables), %d old.", totalN, dur, newN, newSharedN, oldN)
	}()

	ssc.rw.RLock()
	current = make(map[string]*pgStatMonitorExtended, len(ssc.items))
	prev = make(map[string]*pgStatMonitorExtended, len(ssc.items))
	for k, v := range ssc.items {
		prev[k] = v
	}
	ssc.rw.RUnlock()

	// load all databases and usernames first as we can't use querier while iterating over rows below
	databases := queryDatabases(q)
	usernames := queryUsernames(q)

	rows, err := q.SelectRows(pgStatMonitorView, "WHERE queryid IS NOT NULL AND query IS NOT NULL")
	if err != nil {
		err = errors.Wrap(err, "failed to query pg_stat_monitor")
		return
	}
	defer rows.Close() //nolint:errcheck

	for ctx.Err() == nil {
		var row pgStatMonitor
		if err = q.NextRow(&row, rows); err != nil {
			if err == reform.ErrNoRows {
				err = nil
			}
			break
		}
		totalN++

		c := &pgStatMonitorExtended{
			pgStatMonitor: row,
			Database:      databases[row.DBID],
			Username:      usernames[row.UserID],
		}

		if p := prev[c.QueryID]; p != nil {
			oldN++

			c.Fingerprint = p.Fingerprint
			c.Example = p.Example
			c.IsQueryTruncated = p.IsQueryTruncated
		} else {
			newN++

			fingerprint := c.Query
			example := ""
			if !normalizedQuery {
				fingerprint = query.Fingerprint(c.Query)
				example = c.Query
			}
			var isTruncated bool

			c.Fingerprint, isTruncated = truncate.Query(fingerprint)
			if isTruncated {
				c.IsQueryTruncated = isTruncated
			}
			c.Example, isTruncated = truncate.Query(example)
			if isTruncated {
				c.IsQueryTruncated = isTruncated
			}
		}

		current[c.QueryID] = c
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}

	if err != nil {
		err = errors.Wrap(err, "failed to fetch pg_stat_monitor")
	}
	return
}

// stats returns statMonitorCache statistics.
func (ssc *statMonitorCache) stats() statMonitorCacheStats {
	ssc.rw.RLock()
	defer ssc.rw.RUnlock()

	oldest := time.Now().Add(retainStatMonitor)
	var newest time.Time
	for _, t := range ssc.added {
		if oldest.After(t) {
			oldest = t
		}
		if newest.Before(t) {
			newest = t
		}
	}

	return statMonitorCacheStats{
		current:  uint(len(ssc.added)),
		updatedN: ssc.updatedN,
		addedN:   ssc.addedN,
		removedN: ssc.removedN,
		oldest:   oldest,
		newest:   newest,
	}
}

// refresh removes expired items in cache, then adds current items.
func (ssc *statMonitorCache) refresh(current map[string]*pgStatMonitorExtended) {
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
