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

package proto_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/percona/go-mysql/event"
	"github.com/percona/pmm/proto"
	qp "github.com/percona/pmm/proto/qan"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct {
}

var _ = Suite(&TestSuite{})

func (s *TestSuite) TestGetAgentData(t *C) {
	start, _ := time.Parse("2006-01-02 15:04", "2014-11-18 20:24")
	end := start.Add(1 * time.Minute)

	expectedData := qp.Report{
		UUID:    "313",
		StartTs: start,
		EndTs:   end,
		RunTime: 0.007831957,
		Global: &event.Class{
			TotalQueries:  54,
			UniqueQueries: 1,
			Metrics: &event.Metrics{
				TimeMetrics: map[string]*event.TimeStats{
					"Lock_time": &event.TimeStats{
						Sum: 0.006123000021034386,
						Min: 7.300000288523734e-05,
						Avg: 0.00011338888927841456,
						P95: 0.00015300000086426735,
						Med: 0.00011000000085914508,
						Max: 0.00017600000137463212,
					},
					"Query_time": &event.TimeStats{
						Sum: 0.09068399993702769,
						Min: 0.0011579999700188637,
						Avg: 0.0016793333321671795,
						P95: 0.0023590000346302986,
						Med: 0.0017010000301524997,
						Max: 0.002704000100493431,
					},
				},
				NumberMetrics: map[string]*event.NumberStats{
					"Rows_examined": &event.NumberStats{
						Sum: 16848,
						Min: 312,
						Avg: 312,
						P95: 312,
						Med: 312,
						Max: 312,
					},
					"Rows_sent": &event.NumberStats{
						Sum: 16848,
						Min: 312,
						Avg: 312,
						P95: 312,
						Med: 312,
						Max: 312,
					},
				},
			},
		},
		Class: []*event.Class{
			{
				Id:          "B90978440CC11CC7",
				Fingerprint: "show /*!? global */ status",
				Metrics: &event.Metrics{
					TimeMetrics: map[string]*event.TimeStats{
						"Lock_time": &event.TimeStats{
							Sum: 0.006123000021034386,
							Min: 7.300000288523734e-05,
							Avg: 0.00011338888927841456,
							P95: 0.00015300000086426735,
							Med: 0.00011000000085914508,
							Max: 0.00017600000137463212,
						},
						"Query_time": &event.TimeStats{
							Sum: 0.09068399993702769,
							Min: 0.0011579999700188637,
							Avg: 0.0016793333321671795,
							P95: 0.0023590000346302986,
							Med: 0.0017010000301524997,
							Max: 0.002704000100493431,
						},
					},
					NumberMetrics: map[string]*event.NumberStats{
						"Rows_examined": &event.NumberStats{
							Sum: 16848,
							Min: 312,
							Avg: 312,
							P95: 312,
							Med: 312,
							Max: 312,
						},
						"Rows_sent": &event.NumberStats{
							Sum: 16848,
							Min: 312,
							Avg: 312,
							P95: 312,
							Med: 312,
							Max: 312,
						},
					},
				},
				TotalQueries: 54,
				Example: &event.Example{
					QueryTime: 0.002704000100493431,
					Db:        "",
					Query:     "SHOW /*!50002 GLOBAL */ STATUS",
					Ts:        "2014-11-18 17:24:00",
				},
			},
		},
		SlowLogFile: "/var/lib/mysql/karl-Lenovo-G580-slow.log",
		StartOffset: 104807579,
		EndOffset:   104819521,
		StopOffset:  104819522,
	}

	sz := proto.NewJsonGzipSerializer()
	b, err := sz.ToBytes(expectedData)

	data := &proto.Data{
		Hostname:        "Ripley",
		ContentType:     "application/json",
		ContentEncoding: "gzip",
		Data:            b,
	}

	decoded, err := data.GetData()
	t.Assert(err, IsNil)

	var receivedData qp.Report
	_ = json.Unmarshal(decoded, &receivedData)
	t.Check(expectedData, DeepEquals, receivedData)
}
