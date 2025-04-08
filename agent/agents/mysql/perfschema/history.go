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

package perfschema

import (
	"database/sql"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/agent/agents/cache"
	"github.com/percona/pmm/agent/utils/mysql"
)

// historyCache is a wrapper for cache.Cache to use only with historyMap type.
type historyCache struct {
	cache *cache.Cache
}

func (c *historyCache) Set(src historyMap) error {
	return c.cache.Set(src)
}

func (c *historyCache) Get(dest historyMap) error {
	return c.cache.Get(dest)
}

func newHistoryCache(typ historyMap, retain time.Duration, sizeLimit uint, l *logrus.Entry) (*historyCache, error) {
	c, err := cache.New(typ, retain, sizeLimit, l)
	return &historyCache{c}, err
}

func getHistory(q *reform.Querier, long *bool) (historyMap, error) {
	view := eventsStatementsHistoryView
	if long != nil && *long {
		view = eventsStatementsHistoryLongView
	}
	rows, err := q.SelectRows(view, "WHERE DIGEST IS NOT NULL AND SQL_TEXT IS NOT NULL")
	if err != nil {
		return nil, errors.Wrap(err, "failed to query events_statements_history")
	}
	defer rows.Close() //nolint:errcheck

	return getHistoryRows(rows, q)
}

func getHistoryRows(rows *sql.Rows, q *reform.Querier) (historyMap, error) {
	var err error
	res := make(historyMap)
	for {
		var esh eventsStatementsHistory
		if err = q.NextRow(&esh, rows); err != nil {
			break
		}
		res[mysql.QueryIDWithSchema(pointer.GetString(esh.CurrentSchema), *esh.Digest)] = &esh
	}
	if !errors.Is(err, reform.ErrNoRows) {
		return nil, errors.Wrap(err, "failed to fetch events_statements_history")
	}
	return res, nil
}
