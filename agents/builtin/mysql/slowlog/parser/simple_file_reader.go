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

package parser

import (
	"bufio"
	"io"
	"os"
	"sync"
)

// SimpleFileReader reads lines from the single file from the start until EOF.
type SimpleFileReader struct {
	// file Read/Close calls must be synchronized
	m sync.Mutex
	f *os.File
	r *bufio.Reader
}

// NewSimpleFileReader creates new SimpleFileReader.
func NewSimpleFileReader(filename string) (*SimpleFileReader, error) {
	f, err := os.Open(filename) //nolint:gosec
	if err != nil {
		return nil, err
	}
	return &SimpleFileReader{
		f: f,
		r: bufio.NewReader(f),
	}, nil
}

// NextLine implements Reader interface.
func (r *SimpleFileReader) NextLine() (string, error) {
	// TODO handle partial line reads as in ContinuousFileReader if needed

	r.m.Lock()
	l, err := r.r.ReadString('\n')
	r.m.Unlock()
	return l, err
}

// Close implements Reader interface.
func (r *SimpleFileReader) Close() error {
	r.m.Lock()
	defer r.m.Unlock()

	return r.f.Close()
}

// Metrics implements Reader interface.
func (r *SimpleFileReader) Metrics() *ReaderMetrics {
	r.m.Lock()
	defer r.m.Unlock()

	var m ReaderMetrics
	fi, err := r.f.Stat()
	if err == nil {
		m.InputSize = fi.Size()
	}
	pos, err := r.f.Seek(0, io.SeekCurrent)
	if err == nil {
		m.InputPos = pos
	}
	return &m
}

// check interfaces
var (
	_ Reader = (*SimpleFileReader)(nil)
)
