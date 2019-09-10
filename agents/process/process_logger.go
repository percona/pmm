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

package process

import (
	"bytes"
	"io"
	"strings"
	"sync"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
)

// processLogger is a Writer that logs full lines and keeps several latest lines in memory.
type processLogger struct {
	l *logrus.Entry

	m    sync.RWMutex
	buf  []byte
	i    int
	data []*string
}

// newProcessLogger creates new processLogger with a given logger and a given amount of lines to keep.
func newProcessLogger(l *logrus.Entry, lines int) *processLogger {
	return &processLogger{
		l:    l,
		data: make([]*string, lines),
	}
}

// Write implements io.Writer.
// This method is thread-safe.
func (pl *processLogger) Write(p []byte) (n int, err error) {
	pl.m.Lock()
	defer pl.m.Unlock()

	b := bytes.NewBuffer(pl.buf)
	n, err = b.Write(p)
	if err != nil {
		return
	}

	var line string
	for {
		line, err = b.ReadString('\n')
		if err != nil {
			pl.buf = []byte(line)
			err = nil
			return
		}
		line = strings.TrimSuffix(line, "\n")
		if pl.l != nil {
			pl.l.Infoln(line)
		}
		pl.data[pl.i] = pointer.ToString(line)
		pl.i = (pl.i + 1) % len(pl.data)
	}
}

// Latest returns kept lines.
// This method is thread-safe.
func (pl *processLogger) Latest() []string {
	pl.m.RLock()
	defer pl.m.RUnlock()

	result := make([]string, 0, len(pl.data))
	for i := pl.i; i < pl.i+len(pl.data); i++ {
		line := pl.data[i%len(pl.data)]
		if line != nil {
			result = append(result, *line)
		}
	}
	return result
}

// check interfaces
var (
	_ io.Writer = (*processLogger)(nil)
)
