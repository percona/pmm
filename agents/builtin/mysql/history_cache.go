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

package mysql

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
)

const (
	refreshHistoryCache  = time.Second
	cleanupHistoryCache  = 10 * time.Second
	retainInHistoryCache = 5 * time.Minute
)

// historyCache provides cached access to performance_schema.events_statements_history.
type historyCache struct {
	q *reform.Querier
	l *logrus.Entry

	// maps key is digest
	rw     sync.RWMutex
	events map[string]*EventsStatementsHistory
	added  map[string]time.Time
}

// newHistoryCache creates new historyCache.
//
// Caller should call run method.
func newHistoryCache(q *reform.Querier, l *logrus.Entry) *historyCache {
	return &historyCache{
		q:      q,
		l:      l,
		events: make(map[string]*EventsStatementsHistory),
		added:  make(map[string]time.Time),
	}
}

// run runs cache refresher and cleaner until context is canceled.
func (hc *historyCache) run(ctx context.Context) {
	refresher := time.NewTicker(refreshHistoryCache)
	cleaner := time.NewTicker(cleanupHistoryCache)
	defer refresher.Stop()
	defer cleaner.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-refresher.C:
			hc.refresh()
		case <-cleaner.C:
			hc.cleanup()
		}
	}
}

// get returns event by digest from the cache or from performance schema.
func (hc *historyCache) get(digest string) *EventsStatementsHistory {
	// fast path - event is already in cache
	hc.rw.RLock()
	if e := hc.events[digest]; e != nil {
		hc.rw.RUnlock()
		return e
	}

	// refresh cache once and try again
	hc.rw.RUnlock()
	hc.refresh()
	hc.rw.RLock()
	e := hc.events[digest]
	hc.rw.RUnlock()
	return e
}

// refresh adds/overwrites events in cache.
func (hc *historyCache) refresh() {
	structs, err := hc.q.SelectAllFrom(EventsStatementsHistoryView, "WHERE DIGEST IS NOT NULL")
	if err != nil {
		hc.l.Error(err)
		return
	}

	hc.rw.Lock()
	defer hc.rw.Unlock()

	now := time.Now()
	for _, str := range structs {
		esh := str.(*EventsStatementsHistory)
		hc.events[*esh.Digest] = esh
		hc.added[*esh.Digest] = now
	}
}

// cleanup removes old events from cache.
func (hc *historyCache) cleanup() {
	hc.rw.Lock()
	defer hc.rw.Unlock()

	now := time.Now()
	var toDelete []string
	for digest := range hc.events {
		if hc.added[digest].Before(now.Add(-retainInHistoryCache)) {
			toDelete = append(toDelete, digest)
		}
	}

	hc.l.Debugf("Deleting %d out of %d events from history cache.", len(toDelete), len(hc.events))
	for _, digest := range toDelete {
		delete(hc.events, digest)
		delete(hc.added, digest)
	}
}
