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

package metrics

import (
	"fmt"

	"github.com/percona/pmm/proto"
)

var StatNames []string = []string{
	"sum",
	"min",
	"p5", // 5th percentile
	"avg",
	"med", // 50th percentile
	"p95", // 95th percentile
	"max",
}

type Stats struct {
	Cnt uint64
	Sum proto.NullFloat64
	Min proto.NullFloat64
	P5  proto.NullFloat64
	Avg proto.NullFloat64
	Med proto.NullFloat64
	P95 proto.NullFloat64
	Max proto.NullFloat64
}

// AggregateFunction returns a SQL column expression for aggregating the given
// metric and stat. cntCol is used to aggregate averages: sum / cnt.
// For example, to aggregate metric Query_time and stat min: MIN(Query_time_min).
func AggregateFunction(metric, stat, cntCol string) string {
	switch stat {
	case "cnt", "sum":
		return fmt.Sprintf("SUM(%s_%s)", metric, stat)
	case "min":
		return fmt.Sprintf("MIN(%s_%s)", metric, stat)
	case "max":
		return fmt.Sprintf("MAX(%s_%s)", metric, stat)
	case "med", "median", "p5", "p95", "pct_95":
		return fmt.Sprintf("AVG(%s_%s)", metric, stat)
	case "avg":
		return fmt.Sprintf("SUM(%s_sum)/SUM(%s)", metric, cntCol)
	default:
		return ""
	}
}

const (
	UNIVERSAL      = 1
	PERF_SCHEMA    = 2
	PERCONA_SERVER = 4
	META           = 8
	MICROSECOND    = 16
	COUNTER        = 32
)

type MetricFlags struct {
	Name  string
	Flags int
}

// ORDER IS SIGNIFICANT! Database handlers select and scan metrics in order.
var Query []MetricFlags = []MetricFlags{
	MetricFlags{Name: "Query_time", Flags: UNIVERSAL | MICROSECOND},
	MetricFlags{Name: "Lock_time", Flags: UNIVERSAL | MICROSECOND},
	MetricFlags{Name: "Rows_sent", Flags: UNIVERSAL},
	MetricFlags{Name: "Rows_examined", Flags: UNIVERSAL},

	MetricFlags{Name: "Rows_affected", Flags: PERCONA_SERVER | PERF_SCHEMA},
	MetricFlags{Name: "Bytes_sent", Flags: PERCONA_SERVER},
	MetricFlags{Name: "Tmp_tables", Flags: PERCONA_SERVER},
	MetricFlags{Name: "Tmp_disk_tables", Flags: PERCONA_SERVER},
	MetricFlags{Name: "Tmp_table_sizes", Flags: PERCONA_SERVER},
	MetricFlags{Name: "QC_Hit", Flags: PERCONA_SERVER | COUNTER},
	MetricFlags{Name: "Full_scan", Flags: PERCONA_SERVER | PERF_SCHEMA | COUNTER},
	MetricFlags{Name: "Full_join", Flags: PERCONA_SERVER | PERF_SCHEMA | COUNTER},
	MetricFlags{Name: "Tmp_table", Flags: PERCONA_SERVER | PERF_SCHEMA | COUNTER},
	MetricFlags{Name: "Tmp_table_on_disk", Flags: PERCONA_SERVER | PERF_SCHEMA | COUNTER},
	MetricFlags{Name: "Filesort", Flags: PERCONA_SERVER | COUNTER},
	MetricFlags{Name: "Filesort_on_disk", Flags: PERCONA_SERVER | COUNTER},
	MetricFlags{Name: "Merge_passes", Flags: PERCONA_SERVER | PERF_SCHEMA},
	MetricFlags{Name: "InnoDB_IO_r_ops", Flags: PERCONA_SERVER},
	MetricFlags{Name: "InnoDB_IO_r_bytes", Flags: PERCONA_SERVER},
	MetricFlags{Name: "InnoDB_IO_r_wait", Flags: PERCONA_SERVER | MICROSECOND},
	MetricFlags{Name: "InnoDB_rec_lock_wait", Flags: PERCONA_SERVER | MICROSECOND},
	MetricFlags{Name: "InnoDB_queue_wait", Flags: PERCONA_SERVER | MICROSECOND},
	MetricFlags{Name: "InnoDB_pages_distinct", Flags: PERCONA_SERVER},

	MetricFlags{Name: "Errors", Flags: PERF_SCHEMA | COUNTER},
	MetricFlags{Name: "Warnings", Flags: PERF_SCHEMA | COUNTER},
	MetricFlags{Name: "Select_full_range_join", Flags: PERF_SCHEMA | COUNTER},
	MetricFlags{Name: "Select_range", Flags: PERF_SCHEMA | COUNTER},
	MetricFlags{Name: "Select_range_check", Flags: PERF_SCHEMA | COUNTER},
	MetricFlags{Name: "Sort_range", Flags: PERF_SCHEMA | COUNTER},
	MetricFlags{Name: "Sort_rows", Flags: PERF_SCHEMA | COUNTER},
	MetricFlags{Name: "Sort_scan", Flags: PERF_SCHEMA | COUNTER},
	MetricFlags{Name: "No_index_used", Flags: PERF_SCHEMA | COUNTER},
	MetricFlags{Name: "No_good_index_used", Flags: PERF_SCHEMA | COUNTER},

	MetricFlags{Name: "Query_length", Flags: META},
}
