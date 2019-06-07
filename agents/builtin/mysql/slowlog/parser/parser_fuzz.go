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

// +build gofuzz

// See https://github.com/dvyukov/go-fuzz

package parser

import (
	"bufio"
	"bytes"
	"io"

	"github.com/percona/go-mysql/log"
)

type bytesReader struct {
	r *bufio.Reader
}

func newBytesReader(b []byte) (*bytesReader, error) {
	return &bytesReader{
		r: bufio.NewReader(bytes.NewReader(b)),
	}, nil
}

func (r *bytesReader) NextLine() (string, error) {
	return r.r.ReadString('\n')
}

func (r *bytesReader) Close() error {
	panic("not reached")
}

func (r *bytesReader) Metrics() *ReaderMetrics {
	panic("not reached")
}

func Fuzz(data []byte) int {
	r, err := newBytesReader(data)
	if err != nil {
		panic(err)
	}
	p := NewSlowLogParser(r, log.Options{})

	go p.Run()

	for p.Parse() != nil {
	}

	if p.Err() == io.EOF {
		return 1
	}
	return 0
}
