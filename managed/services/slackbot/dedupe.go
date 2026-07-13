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

package slackbot

import (
	"strings"
	"sync"
)

// ringDedupe drops oldest keys when full so memory stays bounded (Slack retries / duplicate deliveries share the same event ts).
type ringDedupe struct {
	mu sync.Mutex
	q  []string
	m  map[string]struct{}
	n  int
}

func newRingDedupe(capacity int) *ringDedupe {
	return &ringDedupe{m: make(map[string]struct{}), n: capacity}
}

// firstSeen returns true the first time this key is observed; false on repeats while the key is still in the ring.
func (r *ringDedupe) firstSeen(parts ...string) bool {
	key := strings.Join(parts, "\x00")
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[key]; ok {
		return false
	}
	r.m[key] = struct{}{}
	r.q = append(r.q, key)
	if len(r.q) > r.n {
		old := r.q[0]
		r.q = r.q[1:]
		delete(r.m, old)
	}
	return true
}

// forget removes a key so a failed turn (e.g. PostMessage error) can be retried with the same Slack message ts.
func (r *ringDedupe) forget(parts ...string) {
	key := strings.Join(parts, "\x00")
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[key]; !ok {
		return
	}
	delete(r.m, key)
	out := r.q[:0]
	for _, k := range r.q {
		if k != key {
			out = append(out, k)
		}
	}
	r.q = out
}

var slackEventDedupe = newRingDedupe(4096) //nolint:mnd
