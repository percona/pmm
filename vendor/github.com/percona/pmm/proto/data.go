/*
   Copyright (c) 2016, Percona LLC and/or its affiliates. All rights reserved.

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package proto

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Data from a tool
type Data struct {
	// Agent sets:
	ProtocolVersion string
	Created         time.Time // when Data was spooled (UTC)
	Hostname        string    // OS instance name the agent is running on
	Service         string    // which tool
	ContentType     string    // of Data ("application/json")
	ContentEncoding string    // of Data ("gzip" or empty)
	Data            []byte    // encoded tool data
}

// In go-mysql/event/query_class.go MAX_EXAMPLE_BYTES=1024*10 (10 KiB) and
// default top query limit is 200 which = ~2M. Then add overhead for JSON
// and a single data msg should not exceed 5 MiB which very liberal because
// the real-world avg is about 300 KiB/msg for QAN (30 KiB/msg for MM).
const MAX_DATA_SIZE = 1024 * 1024 * 5

func (d *Data) GetData() ([]byte, error) {
	if d.Data == nil {
		return nil, nil
	}

	if d.ContentEncoding != "gzip" {
		return d.Data, nil
	}

	// Put the data in a bytes.Buffer because that implements the io.Reader
	// interface that gzip and io.Copy use.
	b := bytes.NewBuffer(d.Data)
	g, err := gzip.NewReader(b)
	if err != nil {
		return nil, fmt.Errorf("Error decompressing gzipped data: %s", err)
	}
	defer g.Close()

	// Read, decompress, and copy into another bytes.Buffer. Limit max read to
	// 10x the max data size because gzip data mangled just right will not
	// trigger an io.EOF so an unbounded copy will read forever.
	unzippedData := &bytes.Buffer{}
	_, err = io.CopyN(unzippedData, g, MAX_DATA_SIZE*10)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return unzippedData.Bytes(), nil
}

type Response struct {
	Code  uint   // standard HTTP status (http://httpstatus.es/)
	Error string // empty if ok (Code=200)
}

type DataSpoolLimits struct {
	MaxAge   uint   // seconds
	MaxSize  uint64 // bytes
	MaxFiles uint
}

type Serializer interface {
	ToBytes(data interface{}) ([]byte, error)
	Encoding() string
	Concurrent() bool
}

type JsonGzipSerializer struct {
	e *json.Encoder
	g *gzip.Writer
	b *bytes.Buffer
}

func NewJsonGzipSerializer() *JsonGzipSerializer {
	b := &bytes.Buffer{}    // 4. buffer
	g := gzip.NewWriter(b)  // 3. gzip
	e := json.NewEncoder(g) // 2. encode
	// ....................... 1. data

	s := &JsonGzipSerializer{
		e: e,
		g: g,
		b: b,
	}
	return s
}

func (s *JsonGzipSerializer) ToBytes(data interface{}) ([]byte, error) {
	s.b.Reset()
	s.g.Reset(s.b)
	if err := s.e.Encode(data); err != nil {
		return nil, err
	}
	s.g.Close()
	return s.b.Bytes(), nil
}

func (s *JsonGzipSerializer) Encoding() string {
	return "gzip"
}

func (s *JsonGzipSerializer) Concurrent() bool {
	return false
}

// --------------------------------------------------------------------------

type JsonSerializer struct {
}

func NewJsonSerializer() *JsonSerializer {
	j := &JsonSerializer{}
	return j
}

func (j *JsonSerializer) ToBytes(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

func (s *JsonSerializer) Encoding() string {
	return ""
}

func (s *JsonSerializer) Concurrent() bool {
	return true
}
