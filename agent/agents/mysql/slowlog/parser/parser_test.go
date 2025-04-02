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

package parser

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/percona/go-mysql/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateF = flag.Bool("update", false, "update golden .json files")

func parseSlowLog(t *testing.T, filepath string, opts log.Options) []log.Event {
	t.Helper()

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
	t.Parallel()

	files, err := filepath.Glob(filepath.FromSlash("./testdata/*.log"))
	require.NoError(t, err)
	for _, file := range files {
		goldenFile := strings.TrimSuffix(file, ".log") + ".json"
		name := strings.TrimSuffix(filepath.Base(file), ".log")
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			opts := log.Options{
				DefaultLocation: time.UTC,
			}
			actual := parseSlowLog(t, file, opts)

			if *updateF {
				b, err := json.MarshalIndent(actual, "", "  ")
				require.NoError(t, err)
				b = append(b, '\n')
				err = os.WriteFile(goldenFile, b, 0o666) //nolint:gosec
				require.NoError(t, err)
				t.Skipf("%s updated.", goldenFile)
			}

			b, err := os.ReadFile(goldenFile) //nolint:gosec
			require.NoError(t, err)
			var expected []log.Event
			err = json.Unmarshal(b, &expected)
			require.NoError(t, err)

			assert.Equal(t, expected, actual)
		})
	}
}

func TestParseTime(t *testing.T) {
	cases := []struct {
		description string
		input       string
		expected    time.Time
	}{
		{
			description: "Should be parsed `Time: 180214 16:18:07`",
			input:       "Time: 180214 16:18:07",
			expected:    time.Date(2018, time.February, 14, 16, 18, 7, 0, time.Local),
		},
		{
			description: "Should be parsed `Time: 280214 16:18:07`",
			input:       "Time: 280214 16:18:07",
			expected:    time.Date(2028, time.February, 14, 16, 18, 7, 0, time.Local),
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			p := NewSlowLogParser(nil, log.Options{})
			p.parseTime(tc.input)
			actual := p.event.Ts
			if actual != tc.expected {
				t.Fatalf("expected: %s got: %s for input: %s", tc.expected, actual, tc.input)
			}
		})
	}
}

func TestParseUser(t *testing.T) {
	type Expected struct {
		UserName string
		Host     string
	}

	cases := []struct {
		description string
		input       string
		expected    Expected
	}{
		{
			description: "Should be parsed `User@Host: bookblogs[bookblogs] @ localhost []  Id: 56601`",
			input:       "User@Host: bookblogs[bookblogs] @ localhost []  Id: 56601",
			expected: Expected{
				UserName: "bookblogs",
				Host:     "localhost",
			},
		},
		{
			description: "Should be parsed `User@Host: user[user] @ some_host []  Id: 56601`",
			input:       "User@Host: user[user] @ some_host []  Id: 56601",
			expected: Expected{
				UserName: "user",
				Host:     "some_host",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			p := NewSlowLogParser(nil, log.Options{})
			p.parseUser(tc.input)
			actualEvent := p.event
			if actualEvent.User != tc.expected.UserName || actualEvent.Host != tc.expected.Host {
				t.Fatalf("expected: %s and %s got: %s and %s for input: %s",
					tc.expected.UserName, tc.expected.Host, actualEvent.User, actualEvent.Host, tc.input)
			}
		})
	}
}

func TestParseMetrics(t *testing.T) {
	const (
		TimeMetrics int = iota
		NumberMetrics
		BoolMetrics
		DB
		Rate
	)

	type Expected struct {
		Type          int
		TimeMetrics   map[string]float64
		NumberMetrics map[string]uint64
		BoolMetrics   map[string]bool
		DB            string
		RateType      string
		RateLimit     uint
	}

	cases := []struct {
		description string
		input       string
		expected    Expected
	}{
		{
			description: "Should be parsed `Query_time: 1.000249  Lock_time: 0.000000`",
			input:       "Query_time: 1.000249  Lock_time: 0.000000",
			expected: Expected{
				Type: TimeMetrics,
				TimeMetrics: map[string]float64{
					"Query_time": 1.000249,
					"Lock_time":  0,
				},
			},
		},
		{
			description: "Should be parsed `Rows_sent: 1  Rows_examined: 0  Rows_affected: 0`",
			input:       "Rows_sent: 1  Rows_examined: 0  Rows_affected: 0",
			expected: Expected{
				Type: NumberMetrics,
				NumberMetrics: map[string]uint64{
					"Rows_sent":     1,
					"Rows_examined": 0,
					"Rows_affected": 0,
				},
			},
		},
		{
			description: "Should be parsed `QC_Hit: No  Full_scan: Yes  Full_join: No  Tmp_table: No  Tmp_table_on_disk: No`",
			input:       "QC_Hit: No  Full_scan: Yes  Full_join: No  Tmp_table: No  Tmp_table_on_disk: No",
			expected: Expected{
				Type: BoolMetrics,
				BoolMetrics: map[string]bool{
					"QC_Hit":            false,
					"Full_scan":         true,
					"Full_join":         false,
					"Tmp_table":         false,
					"Tmp_table_on_disk": false,
				},
			},
		},
		{
			description: "Should be parsed `Schema: maindb  Last_errno: 0  Killed: 0`",
			input:       "Schema: maindb  Last_errno: 0  Killed: 0",
			expected: Expected{
				Type: DB,
				DB:   "maindb",
			},
		},
		{
			description: "Should be parsed `Log_slow_rate_type: query  Log_slow_rate_limit: 2`",
			input:       "Log_slow_rate_type: query  Log_slow_rate_limit: 2",
			expected: Expected{
				Type:      Rate,
				RateType:  "query",
				RateLimit: uint(2),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			p := NewSlowLogParser(nil, log.Options{})
			p.parseMetrics(tc.input)
			actualEvent := p.event

			switch tc.expected.Type {
			case TimeMetrics:
				if !reflect.DeepEqual(actualEvent.TimeMetrics, tc.expected.TimeMetrics) {
					t.Fatalf("expected: %#v got: %#v for input: %s",
						tc.expected.TimeMetrics, actualEvent.TimeMetrics, tc.input)
				}
			case NumberMetrics:
				if !reflect.DeepEqual(actualEvent.NumberMetrics, tc.expected.NumberMetrics) {
					t.Fatalf("expected: %#v got: %#v for input: %s",
						tc.expected.NumberMetrics, actualEvent.NumberMetrics, tc.input)
				}
			case BoolMetrics:
				if !reflect.DeepEqual(actualEvent.BoolMetrics, tc.expected.BoolMetrics) {
					t.Fatalf("expected: %#v got: %#v for input: %s",
						tc.expected.BoolMetrics, actualEvent.BoolMetrics, tc.input)
				}
			case DB:
				if actualEvent.Db != tc.expected.DB {
					t.Fatalf("expected: %s got: %s for input: %s",
						tc.expected.DB, actualEvent.Db, tc.input)
				}
			case Rate:
				if actualEvent.RateLimit != tc.expected.RateLimit || actualEvent.RateType != tc.expected.RateType {
					t.Fatalf("expected %d and %s got: %d and %s for input: %s",
						tc.expected.RateLimit, tc.expected.RateType,
						actualEvent.RateLimit, actualEvent.RateType, tc.input)
				}
			}
		})
	}
}

func TestParserSpecial(t *testing.T) {
	t.Parallel()

	t.Run("slow009/FilterAdminCommands", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

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
