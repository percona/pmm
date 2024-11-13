// Copyright (C) 2023 Percona LLC
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
	"regexp"
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

	replacer *strings.Replacer
}

// newProcessLogger creates new processLogger with a given logger and a given amount of lines to keep.
func newProcessLogger(l *logrus.Entry, lines int, redactWords []string) *processLogger {
	pl := &processLogger{
		l:    l,
		data: make([]*string, lines),
	}

	if l != nil && l.Logger.GetLevel() >= logrus.DebugLevel {
		l.Debug("Logs redactor disabled in debug mode.")
	} else {
		pl.replacer = replacer(redactWords)
	}

	return pl
}

// Write implements io.Writer.
// This method is thread-safe.
func (pl *processLogger) Write(p []byte) (int, error) {
	pl.m.Lock()
	defer pl.m.Unlock()

	b := bytes.NewBuffer(pl.buf)
	n, err := b.Write(p)
	if err != nil {
		return n, err
	}

	var line string
	for {
		line, err = b.ReadString('\n')
		if err != nil {
			pl.buf = []byte(line)
			return n, nil //nolint:nilerr
		}
		line = strings.TrimSuffix(line, "\n")
		if pl.replacer != nil {
			line = pl.replacer.Replace(line)
		}
		if pl.l != nil {
			level, found, err := extractLogLevel(line)

			switch {
			case err != nil:
				pl.l.Warnf("Extract log level error: %v.", err)
				pl.l.Infoln(line)
			case found:
				if level < logrus.ErrorLevel {
					level = logrus.ErrorLevel
				}

				pl.l.Logln(level, line)
			default:
				pl.l.Infoln(line)
			}
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

func replacer(redactWords []string) *strings.Replacer {
	if len(redactWords) == 0 {
		return nil
	}

	r := make([]string, 0, len(redactWords)*2)
	for _, w := range redactWords {
		if w == "" {
			panic("redact word can't be empty")
		}
		r = append(r, w, "***")
	}
	return strings.NewReplacer(r...)
}

var extractLogLevelRegex = regexp.MustCompile(`level=(\w+)`)

func extractLogLevel(line string) (logrus.Level, bool, error) {
	matches := extractLogLevelRegex.FindStringSubmatch(line)

	noMatches := len(matches) < 2
	if noMatches {
		return 0, false, nil
	}

	level, err := logrus.ParseLevel(matches[1])
	if err != nil {
		return 0, false, err
	}

	return level, true, nil
}

// check interfaces.
var (
	_ io.Writer = (*processLogger)(nil)
)
