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
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/agent/agents/cache"
	"github.com/percona/pmm/agent/agents/mysql/shared"
)

// summaryCache is a wrapper for cache.Cache to use only with summaryMap type.
type summaryCache struct {
	cache *cache.Cache
}

func (c *summaryCache) Set(src summaryMap) error {
	return c.cache.Set(src)
}

func (c *summaryCache) Get(dest summaryMap) error {
	return c.cache.Get(dest)
}

func newSummaryCache(typ summaryMap, retain time.Duration, sizeLimit uint, l *logrus.Entry) (*summaryCache, error) {
	c, err := cache.New(typ, retain, sizeLimit, l)
	return &summaryCache{c}, err
}

func getSummaries(q *reform.Querier) (summaryMap, error) {
	rows, err := q.SelectRows(eventsStatementsSummaryByDigestView, "WHERE DIGEST IS NOT NULL AND DIGEST_TEXT IS NOT NULL")
	if err != nil {
		return nil, errors.Wrap(err, "failed to query events_statements_summary_by_digest")
	}
	defer rows.Close() //nolint:errcheck

	res := make(summaryMap)
	for {
		var ess eventsStatementsSummaryByDigest
		if err = q.NextRow(&ess, rows); err != nil {
			break
		}

		// From https://dev.mysql.com/doc/relnotes/mysql/8.0/en/news-8-0-11.html:
		// > The Performance Schema could produce DIGEST_TEXT values with a trailing space. […] (Bug #26908015)
		*ess.DigestText = strings.TrimSpace(*ess.DigestText)
		queryID := shared.QueryIDWithSchema(pointer.GetString(ess.SchemaName), *ess.Digest)
		res[queryID] = &ess
	}
	if !errors.Is(err, reform.ErrNoRows) {
		return nil, errors.Wrap(err, "failed to fetch events_statements_summary_by_digest")
	}
	return res, nil
}
