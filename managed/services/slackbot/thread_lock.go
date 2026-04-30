// Copyright (C) 2026 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package slackbot

import "sync"

// threadLocks serializes handleTurn per Slack thread so history + Slack replies stay consistent.
// Mutex entries are never removed (one per distinct thread key); typical Slack thread volume is fine.
var threadLocks sync.Map // ThreadKey -> *sync.Mutex

func acquireThreadLock(k ThreadKey) func() {
	v, _ := threadLocks.LoadOrStore(k, new(sync.Mutex))
	m := v.(*sync.Mutex)
	m.Lock()
	return m.Unlock
}
