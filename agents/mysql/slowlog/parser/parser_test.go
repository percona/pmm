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

package parser

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/percona/go-mysql/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateF = flag.Bool("update", false, "update golden .json files")

func parseSlowLog(t *testing.T, filepath string, opts log.Options) []log.Event {
	r, err := NewSimpleFileReader(filepath)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, r.Close())
	}()

	p := NewSlowLogParser(r, opts)
	go p.Run()

	res := []log.Event{}
	for {
		e := p.Parse()
		if e == nil {
			require.Equal(t, io.EOF, p.Err())
			return res
		}
		res = append(res, *e)
	}
}

func TestParserGolden(t *testing.T) {
	files, err := filepath.Glob(filepath.FromSlash("./testdata/*.log"))
	require.NoError(t, err)
	for _, file := range files {
		file := file
		goldenFile := strings.TrimSuffix(file, ".log") + ".json"
		name := strings.TrimSuffix(filepath.Base(file), ".log")
		t.Run(name, func(t *testing.T) {
			opts := log.Options{
				DefaultLocation: time.UTC,
			}
			actual := parseSlowLog(t, file, opts)

			if *updateF {
				b, err := json.MarshalIndent(actual, "", "  ")
				require.NoError(t, err)
				b = append(b, '\n')
				err = ioutil.WriteFile(goldenFile, b, 0o666)
				require.NoError(t, err)
				t.Skipf("%s updated.", goldenFile)
			}

			b, err := ioutil.ReadFile(goldenFile) //nolint:gosec
			require.NoError(t, err)
			var expected []log.Event
			err = json.Unmarshal(b, &expected)
			require.NoError(t, err)

			assert.Equal(t, expected, actual)
		})
	}
}

func TestParserSpecial(t *testing.T) {
	t.Run("slow009/FilterAdminCommands", func(t *testing.T) {
		opts := log.Options{
			DefaultLocation: time.UTC,
			FilterAdminCommand: map[string]bool{
				"Quit": true,
			},
		}
		actual := parseSlowLog(t, filepath.Join("testdata", "slow009.log"), opts)
		expect := []log.Event{{
			Query:     "Refresh",
			Db:        "",
			Admin:     true,
			Host:      "localhost",
			User:      "root",
			Offset:    196,
			OffsetEnd: 562,
			Ts:        time.Date(2009, 0o3, 11, 18, 11, 50, 0, time.UTC),
			TimeMetrics: map[string]float64{
				"Query_time": 0.017850,
				"Lock_time":  0.000000,
			},
			NumberMetrics: map[string]uint64{
				"Merge_passes":  0,
				"Rows_affected": 0,
				"Rows_examined": 0,
				"Rows_read":     0,
				"Rows_sent":     0,
				"Thread_id":     47,
			},
			BoolMetrics: map[string]bool{
				"QC_Hit":            false,
				"Full_scan":         false,
				"Full_join":         false,
				"Tmp_table":         false,
				"Tmp_table_on_disk": false,
				"Filesort":          false,
				"Filesort_on_disk":  false,
			},
		}}
		assert.Equal(t, expect, actual)
	})

	t.Run("slow023", func(t *testing.T) {
		r, err := NewSimpleFileReader(filepath.Join("testdata", "slow023.log"))
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, r.Close())
		}()

		opts := log.Options{
			DefaultLocation: time.UTC,
		}
		p := NewSlowLogParser(r, opts)
		go p.Run()

		lastQuery := ""
		for {
			e := p.Parse()
			if e == nil {
				require.Equal(t, io.EOF, p.Err())
				return
			}
			if e.Query == "" {
				t.Errorf("Empty query at offset: %d. Last valid query: %s\n", e.Offset, lastQuery)
			} else {
				lastQuery = e.Query
			}
		}
	})
}
