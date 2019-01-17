// pmm-agent
// Copyright (C) 2018 Percona LLC
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

package supervisor

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
