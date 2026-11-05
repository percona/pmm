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

import "sync"

// threadLocks serializes handleTurn per Slack thread so history + Slack replies stay consistent.
// Mutex entries are never removed (one per distinct thread key); typical Slack thread volume is fine.
var threadLocks sync.Map // ThreadKey -> *sync.Mutex

func acquireThreadLock(k ThreadKey) func() {
	v, _ := threadLocks.LoadOrStore(k, new(sync.Mutex))
	m := v.(*sync.Mutex) //nolint:forcetypeassert // we just stored *sync.Mutex above
	m.Lock()
	return m.Unlock
}
