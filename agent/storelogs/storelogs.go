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

// Package storelogs help to store logs
package storelogs

import (
	"container/ring"
	"strings"
	"sync"
)

// LogsStore implement ring save logs.
type LogsStore struct {
	log   *ring.Ring
	count int
	m     sync.Mutex
}

// New creates LogsStore.
func New(count int) *LogsStore {
	return &LogsStore{
		log:   ring.New(count),
		count: count,
	}
}

// Write writes log for store.
func (l *LogsStore) Write(b []byte) (int, error) {
	l.m.Lock()
	defer l.m.Unlock()

	// when store 0 logs
	if l.log == nil {
		return len(b), nil
	}

	l.log.Value = string(b)
	l.log = l.log.Next()
	return len(b), nil
}

// UpdateCount to update max length.
func (l *LogsStore) UpdateCount(count int) {
	l.m.Lock()
	defer l.m.Unlock()

	if l.count == count {
		return
	}

	old := l.log

	l.count = count
	l.log = ring.New(count)
	if l.log == nil {
		return
	}

	old.Do(func(p interface{}) {
		if p != nil {
			l.log.Value = p
			l.log = l.log.Next()
		}
	})
}

// GetLogs return all logs.
func (l *LogsStore) GetLogs() []string {
	l.m.Lock()
	defer l.m.Unlock()

	// when store 0 logs
	if l.log == nil {
		return nil
	}

	logs := make([]string, 0, l.count)

	replacer := strings.NewReplacer("\u001B[36m", "", "\u001B[0m", "", "\u001B[33", "", "\u001B[31m", "", "        ", " ")
	l.log.Do(func(p interface{}) {
		if p != nil {
			logs = append(logs, replacer.Replace(p.(string)))
		}
	})

	return logs
}
