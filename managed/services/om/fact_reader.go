// Copyright (C) 2023 Percona LLC
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

package om

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// observation is one subject's current fact document from om_inventory, with the single collected_at
// that dates every fact in it.
type observation struct {
	Facts       map[string]any `json:"facts"`
	CollectedAt time.Time      `json:"collected_at"`
}

// factReader reads a subject's facts from om_inventory; the start gate recomputes the report digest
// from these fields, so a missing subject or a stale fact must read as absent, never as current.
type factReader struct {
	app extensionsApp
}

// newFactReader builds a reader against om_inventory on the configured Extensions client.
func newFactReader(client *extensionsClient) *factReader {
	return &factReader{app: client.app(probeAppModule)}
}

// Observation reads the current fact document for a subject under a profile; a subject no scan has
// reported yet returns no facts and a zero time rather than an error.
func (r *factReader) Observation(ctx context.Context, subjectKind, subjectKey, profile string) (map[string]any, time.Time, error) {
	query := url.Values{"subject_kind": {subjectKind}, "profile": {profile}}
	out := observation{}
	err := r.app.call(asService(ctx), inventoryCall{
		method: http.MethodGet,
		path:   inventoryPath("observations", subjectKey),
		query:  query,
	}, &out)
	if status.Code(err) == codes.NotFound {
		return nil, time.Time{}, nil
	}
	if err != nil {
		return nil, time.Time{}, err
	}
	return out.Facts, out.CollectedAt, nil
}

// freshFacts drops every fact older than the maximum age its check set declared, so a slow or failed
// scan reads as missing and can never pass for current facts; a fact with no declared age is kept.
func freshFacts(facts map[string]any, collectedAt time.Time, maxAges map[string]time.Duration, now time.Time) map[string]any {
	age := now.Sub(collectedAt)
	fresh := make(map[string]any, len(facts))
	for name, value := range facts {
		maxAge, bounded := maxAges[name]
		if bounded && age > maxAge {
			continue
		}
		fresh[name] = value
	}
	return fresh
}
