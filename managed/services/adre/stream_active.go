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

package adre

import (
	"context"
	"sync"
)

// ActiveChatStreams tracks in-flight PostChat streams per conversation id so DELETE can abort them.
type ActiveChatStreams struct {
	mu sync.Mutex
	m  map[int64]context.CancelFunc
}

// NewActiveChatStreams creates an empty registry.
func NewActiveChatStreams() *ActiveChatStreams {
	return &ActiveChatStreams{m: make(map[int64]context.CancelFunc)}
}

// Register adds a cancel function; the returned callback unregisters when the stream ends.
func (a *ActiveChatStreams) Register(conversationID int64, cancel context.CancelFunc) (done func()) { //nolint:nonamedreturns
	a.mu.Lock()
	a.m[conversationID] = cancel
	a.mu.Unlock()
	return func() {
		a.mu.Lock()
		delete(a.m, conversationID)
		a.mu.Unlock()
	}
}

// Abort cancels an active stream for the conversation, if any.
func (a *ActiveChatStreams) Abort(conversationID int64) {
	a.mu.Lock()
	if c, ok := a.m[conversationID]; ok {
		delete(a.m, conversationID)
		c()
	}
	a.mu.Unlock()
}
