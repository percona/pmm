// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package grafana

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const defaultDashboardCacheTTL = 60 * time.Second

type dashboardJSONCache struct {
	mu    sync.RWMutex
	items map[string]dashboardCacheEntry
	ttl   time.Duration
}

type dashboardCacheEntry struct {
	env *dashboardAPIEnvelope
	exp time.Time
}

func newDashboardJSONCache(ttl time.Duration) *dashboardJSONCache {
	if ttl <= 0 {
		ttl = defaultDashboardCacheTTL
	}
	return &dashboardJSONCache{
		items: make(map[string]dashboardCacheEntry),
		ttl:   ttl,
	}
}

func cacheKey(orgID int, dashboardUID string) string {
	return fmt.Sprintf("%d:%s", orgID, dashboardUID)
}

//nolint:funcorder // private cache primitives grouped together; reads better than visibility ordering
func (c *dashboardJSONCache) get(orgID int, dashboardUID string) (*dashboardAPIEnvelope, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ent, ok := c.items[cacheKey(orgID, dashboardUID)]
	if !ok || time.Now().After(ent.exp) {
		return nil, false
	}
	return ent.env, true
}

//nolint:funcorder // private cache primitives grouped together; reads better than visibility ordering
func (c *dashboardJSONCache) set(orgID int, dashboardUID string, env *dashboardAPIEnvelope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[cacheKey(orgID, dashboardUID)] = dashboardCacheEntry{
		env: env,
		exp: time.Now().Add(c.ttl),
	}
}

// Invalidate drops a cached dashboard entry (e.g. after refresh_dashboard on resolve).
func (c *dashboardJSONCache) Invalidate(orgID int, dashboardUID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, cacheKey(orgID, dashboardUID))
}

func (c *dashboardJSONCache) fetchOrLoad(ctx context.Context, _ *Client, orgID int, dashboardUID string, _ http.Header, loader func(context.Context) (*dashboardAPIEnvelope, error)) (*dashboardAPIEnvelope, error) { //nolint:lll
	if env, ok := c.get(orgID, dashboardUID); ok {
		return env, nil
	}
	env, err := loader(ctx)
	if err != nil {
		return nil, err
	}
	c.set(orgID, dashboardUID, env)
	return env, nil
}

// wrapFetchDashboard wraps fetchDashboardEnvelope for cache.
func wrapFetchDashboard(client *Client, dashboardUID string, headers http.Header) func(context.Context) (*dashboardAPIEnvelope, error) {
	return func(ctx context.Context) (*dashboardAPIEnvelope, error) {
		env, err := fetchDashboardEnvelope(ctx, client, dashboardUID, headers)
		if err != nil {
			return nil, fmt.Errorf("fetch dashboard: %w", err)
		}
		return env, nil
	}
}
