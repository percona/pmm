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

package pgstatstatements

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

func getStatStatements(q *reform.Querier) (map[int64]*pgStatStatements, error) {
	structs, err := q.SelectAllFrom(pgStatStatementsView, "WHERE queryid IS NOT NULL AND query IS NOT NULL")
	if err != nil {
		return nil, errors.Wrap(err, "failed to query pg_stat_statements")
	}

	res := make(map[int64]*pgStatStatements, len(structs))
	for _, str := range structs {
		pss := str.(*pgStatStatements)
		res[*pss.QueryID] = pss
	}
	return res, nil
}

// statStatementCache provides cached access to pg_stat_statements.
// It retains data longer than this table.
type statStatementCache struct {
	retain time.Duration

	rw    sync.RWMutex
	items map[int64]*pgStatStatements
	added map[int64]time.Time
}

// newStatStatementCache creates new statStatementCache.
func newStatStatementCache(retain time.Duration) *statStatementCache {
	return &statStatementCache{
		retain: retain,
		items:  make(map[int64]*pgStatStatements),
		added:  make(map[int64]time.Time),
	}
}

// get returns all current items.
func (c *statStatementCache) get() map[int64]*pgStatStatements {
	c.rw.RLock()
	defer c.rw.RUnlock()

	res := make(map[int64]*pgStatStatements, len(c.items))
	for k, v := range c.items {
		res[k] = v
	}
	return res
}

// refresh removes expired items in cache, then adds current items.
func (c *statStatementCache) refresh(current map[int64]*pgStatStatements) {
	c.rw.Lock()
	defer c.rw.Unlock()

	now := time.Now()

	for k, t := range c.added {
		if now.Sub(t) > c.retain {
			delete(c.items, k)
			delete(c.added, k)
		}
	}

	for k, v := range current {
		c.items[k] = v
		c.added[k] = now
	}
}
