// qan-api2
// Copyright (C) 2019 Percona LLC
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

package analitycs

import (
	"github.com/percona/qan-api2/models"
)

// Service implements gRPC service to communicate with QAN-APP.
type Service struct {
	rm models.Reporter
	mm models.Metrics
}

// NewService create new insstance of Service.
func NewService(rm models.Reporter, mm models.Metrics) *Service {
	return &Service{rm, mm}
}

var standartDimensions = map[string]struct{}{
	"queryid":     {},
	"server":      {},
	"database":    {},
	"schema":      {},
	"username":    {},
	"client_host": {},
}

var boolColumnNames = map[string]struct{}{
	"qc_hit":                 {},
	"full_scan":              {},
	"full_join":              {},
	"tmp_table":              {},
	"tmp_table_on_disk":      {},
	"filesort":               {},
	"filesort_on_disk":       {},
	"select_full_range_join": {},
	"select_range":           {},
	"select_range_check":     {},
	"sort_range":             {},
	"sort_rows":              {},
	"sort_scan":              {},
	"no_index_used":          {},
	"no_good_index_used":     {},
}

//nolint
var specialColumnNames = map[string]struct{}{
	"num_queries": {},
	"load":        {},
}

var commonColumnNames = map[string]struct{}{
	"query_time":            {},
	"lock_time":             {},
	"rows_sent":             {},
	"rows_examined":         {},
	"rows_affected":         {},
	"rows_read":             {},
	"merge_passes":          {},
	"innodb_io_r_ops":       {},
	"innodb_io_r_bytes":     {},
	"innodb_io_r_wait":      {},
	"innodb_rec_lock_wait":  {},
	"innodb_queue_wait":     {},
	"innodb_pages_distinct": {},
	"query_length":          {},
	"bytes_sent":            {},
	"tmp_tables":            {},
	"tmp_disk_tables":       {},
	"tmp_table_sizes":       {},
	"docs_returned":         {},
	"response_length":       {},
	"docs_scanned":          {},
}

func interfaceToFloat32(unk interface{}) float32 {
	switch i := unk.(type) {
	case float64:
		return float32(i)
	case float32:
		return i
	case int64:
		return float32(i)
	default:
		return float32(0)
	}
}
