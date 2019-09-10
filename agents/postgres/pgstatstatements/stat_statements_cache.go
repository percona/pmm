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
	"sync"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

func getStatStatementsExtended(q *reform.Querier) (map[int64]*pgStatStatementsExtended, error) {
	structs, err := q.SelectAllFrom(pgStatStatementsView, "WHERE queryid IS NOT NULL AND query IS NOT NULL")
	if err != nil {
		return nil, errors.Wrap(err, "failed to query pg_stat_statements")
	}

	res := make(map[int64]*pgStatStatementsExtended, len(structs))
	for _, str := range structs {
		pss := str.(*pgStatStatements)
		res[*pss.QueryID] = &pgStatStatementsExtended{
			PgStatStatements: pss,
		}
	}
	return res, nil
}

// statStatementCache provides cached access to pg_stat_statements.
// It retains data longer than this table.
type statStatementCache struct {
	retain time.Duration

	rw    sync.RWMutex
	items map[int64]*pgStatStatementsExtended
	added map[int64]time.Time
}

// newStatStatementCache creates new statStatementCache.
func newStatStatementCache(retain time.Duration) *statStatementCache {
	return &statStatementCache{
		retain: retain,
		items:  make(map[int64]*pgStatStatementsExtended),
		added:  make(map[int64]time.Time),
	}
}

// get returns all current items.
func (c *statStatementCache) get() map[int64]*pgStatStatementsExtended {
	c.rw.RLock()
	defer c.rw.RUnlock()

	res := make(map[int64]*pgStatStatementsExtended, len(c.items))
	for k, v := range c.items {
		res[k] = v
	}
	return res
}

// refresh removes expired items in cache, then adds current items.
func (c *statStatementCache) refresh(current map[int64]*pgStatStatementsExtended) {
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
