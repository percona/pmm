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
	"sync"
)

// ThreadKey identifies a Slack conversation thread (team + channel + root ts).
type ThreadKey struct {
	TeamID    string
	ChannelID string
	ThreadTS  string
}

// ThreadStore holds recent user/assistant messages per thread (RAM only; leader process).
type ThreadStore struct {
	mu    sync.Mutex
	items map[ThreadKey][]any
}

// NewThreadStore creates an empty thread store.
func NewThreadStore() *ThreadStore {
	return &ThreadStore{items: make(map[ThreadKey][]any)}
}

func msgMap(role, content string) map[string]any {
	return map[string]any{"role": role, "content": content}
}

// AppendUser appends a user message and caps length.
func (ts *ThreadStore) AppendUser(key ThreadKey, text string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.items[key] = append(ts.items[key], msgMap("user", text))
	ts.items[key] = trimHistory(ts.items[key], maxThreadMessagesRAM)
}

// UndoLastUserMessage removes the last message if it is a user turn matching content (used when PostMessage fails after append).
func (ts *ThreadStore) UndoLastUserMessage(key ThreadKey, content string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	arr := ts.items[key]
	if len(arr) == 0 {
		return
	}
	last, ok := arr[len(arr)-1].(map[string]any)
	if !ok {
		return
	}
	if role, _ := last["role"].(string); role != "user" {
		return
	}
	if c, _ := last["content"].(string); c != content {
		return
	}
	arr = arr[:len(arr)-1]
	if len(arr) == 0 {
		delete(ts.items, key)
		return
	}
	ts.items[key] = arr
}

// AppendAssistant appends an assistant message and caps length.
func (ts *ThreadStore) AppendAssistant(key ThreadKey, text string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.items[key] = append(ts.items[key], msgMap("assistant", text))
	ts.items[key] = trimHistory(ts.items[key], maxThreadMessagesRAM)
}

// Snapshot returns a shallow copy of messages for ADRE /api/chat conversation_history (oldest first).
func (ts *ThreadStore) Snapshot(key ThreadKey) []any {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	src := ts.items[key]
	out := make([]any, len(src))
	copy(out, src)
	return out
}

func trimHistory(h []any, maxLen int) []any {
	if len(h) <= maxLen {
		return h
	}
	return h[len(h)-maxLen:]
}
