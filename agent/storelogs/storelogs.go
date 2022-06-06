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
	"fmt"
	"strings"
	"sync"
)

// LogsStore implement ring save logs.
type LogsStore struct {
	log *ring.Ring
	m   sync.RWMutex
}

// New creates LogsStore.
func New(count int) *LogsStore {
	return &LogsStore{
		log: ring.New(count),
		m:   sync.RWMutex{},
	}
}

// Write writes log for store.
func (l *LogsStore) Write(b []byte) (n int, err error) {
	l.m.Lock()
	l.log.Value = string(b)
	l.log = l.log.Next()
	l.m.Unlock()
	return len(b), nil
}

// GetLogs return all logs.
func (l *LogsStore) GetLogs() (logs []string) {
	if l != nil {
		l.m.Lock()
		l.log.Do(func(p interface{}) {
			log := fmt.Sprint(p)
			replacer := strings.NewReplacer("\u001B[36m", "", "\u001B[0m", "", "\u001B[33", "", "\u001B[31m", "", "        ", " ")
			if p != nil {
				logs = append(logs, replacer.Replace(log))
			}
		})
		l.m.Unlock()
	}
	return logs
}
