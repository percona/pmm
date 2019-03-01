// qan-api
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

package models

import (
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"net"
	"reflect"

	_ "github.com/kshvakov/clickhouse"
	"github.com/kshvakov/clickhouse/lib/column"
	"github.com/kshvakov/clickhouse/lib/types"
)

// Any is a stub for any argument in SQL query or exec.
type Any struct{}

// Match is a stub method for any argument in SQL query or exec.
func (a Any) Match(v driver.Value) bool {
	return true
}

// TODO: is it can be public and imported?
// https://github.com/kshvakov/clickhouse/blob/4e2869334f3815d257318e813e2627efce90a0d7/value_converter.go#L22
type converter struct{}

func (c *converter) ConvertValue(v interface{}) (driver.Value, error) {
	if driver.IsValue(v) {
		return v, nil
	}

	switch value := v.(type) {
	case int:
		return int64(value), nil
	case int8:
		return int64(value), nil
	case int16:
		return int64(value), nil
	case int32:
		return int64(value), nil
	case int64:
		return value, nil
	case uint:
		return int64(value), nil
	case uint8:
		return int64(value), nil
	case uint16:
		return int64(value), nil
	case uint32:
		return int64(value), nil
	case uint64:
		if value >= 1<<63 {
			v := make([]byte, 8)
			binary.LittleEndian.PutUint64(v, value)
			return v, nil
		}
		return int64(value), nil
	case float32:
		return float64(value), nil
	case float64:
		return value, nil
	case
		[]int, []int8, []int16, []int32, []int64,
		[]uint, []uint8, []uint16, []uint32, []uint64,
		[]float32, []float64,
		[]string:
		return (types.NewArray(v)).Value()
	case net.IP:
		return column.IP(value).Value()
	case driver.Valuer:
		return value.Value()
	}

	switch value := reflect.ValueOf(v); value.Kind() {
	case reflect.Bool:
		if value.Bool() {
			return int64(1), nil
		}
		return int64(0), nil
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int(), nil
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(value.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return value.Float(), nil
	case reflect.String:
		return value.String(), nil
	}

	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, nil
		}
		return c.ConvertValue(rv.Elem().Interface())
	}

	return nil, fmt.Errorf("value converter: unsupported type %T", v)
}

// TODO: Fix tests
// func TestSave(t *testing.T) {
// 	agentMsg := pbqan.AgentMessage{
// 		QueryClass: []*pbqan.QueryClass{
// 			{
// 				Queryid:  "Queryid1",
// 				Labels:   map[string]string{"label1": "aaa1"},
// 				Warnings: map[uint64]uint64{123: 111},
// 				Errors:   map[uint64]uint64{123: 333},
// 			},
// 			{
// 				Queryid:  "Queryid2",
// 				Labels:   map[string]string{"label2": "bbb2"},
// 				Warnings: map[uint64]uint64{321: 222},
// 				Errors:   map[uint64]uint64{321: 444},
// 			},
// 		},
// 	}
// 	var _converter = &converter{}
// 	db, mock, err := sqlmock.New(sqlmock.ValueConverterOption(_converter))
// 	if err != nil {
// 		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
// 	}

// 	mock.ExpectBegin()
// 	a := mock.ExpectPrepare("^INSERT INTO queries .*")
// 	for _, qc := range agentMsg.QueryClass {
// 		s := reflect.ValueOf(*qc)
// 		ret := make([]driver.Value, s.NumField())
// 		for i := 0; i < s.NumField(); i++ {
// 			ret[i] = Any{}
// 		}
// 		ret[0] = qc.Queryid
// 		ret[6], ret[7] = MapToArrsStrStr(qc.Labels)     // Query class labels.
// 		ret[17], ret[18] = MapToArrsIntInt(qc.Warnings) // Query class warnings.
// 		ret[20], ret[21] = MapToArrsIntInt(qc.Errors)   // Query class errors.
// 		a.ExpectExec().WithArgs(ret...).WillReturnResult(sqlmock.NewResult(1, 1))
// 	}
// 	mock.ExpectCommit()
// 	qc := NewQueryClass(sqlx.NewDb(db, "clickhouse"))

// 	// execute save method
// 	if err = qc.Save(&agentMsg); err != nil {
// 		t.Errorf("error was not expected while saving data to clickhouse: %s", err)
// 	}

// 	_ = db.Close()
// 	// we make sure that all expectations were met
// 	if err := mock.ExpectationsWereMet(); err != nil {
// 		t.Errorf("there were unfulfilled expectations: %s", err)
// 	}
// }

// func TestSaveEpmtyMaps(t *testing.T) {
// 	agentMsg := pbqan.AgentMessage{
// 		QueryClass: []*pbqan.QueryClass{
// 			{
// 				Queryid: "Queryid1",
// 			},
// 			{
// 				Queryid: "Queryid2",
// 			},
// 		},
// 	}
// 	var _converter = &converter{}
// 	db, mock, err := sqlmock.New(sqlmock.ValueConverterOption(_converter))
// 	if err != nil {
// 		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
// 	}

// 	mock.ExpectBegin()
// 	a := mock.ExpectPrepare("^INSERT INTO queries .*")
// 	for _, qc := range agentMsg.QueryClass {
// 		s := reflect.ValueOf(*qc)
// 		ret := make([]driver.Value, s.NumField())
// 		for i := 0; i < s.NumField(); i++ {
// 			ret[i] = Any{}
// 		}
// 		ret[0] = qc.Queryid
// 		a.ExpectExec().WithArgs(ret...).WillReturnResult(sqlmock.NewResult(1, 1))
// 	}
// 	mock.ExpectCommit()
// 	qc := NewQueryClass(sqlx.NewDb(db, "clickhouse"))

// 	// execute save method
// 	if err = qc.Save(&agentMsg); err != nil {
// 		t.Errorf("error was not expected while saving data to clickhouse: %s", err)
// 	}

// 	_ = db.Close()
// 	// we make sure that all expectations were met
// 	if err := mock.ExpectationsWereMet(); err != nil {
// 		t.Errorf("there were unfulfilled expectations: %s", err)
// 	}
// }

// func TestSaveEpmtyQueryClass(t *testing.T) {
// 	agentMsg := pbqan.AgentMessage{
// 		QueryClass: []*pbqan.QueryClass{},
// 	}
// 	var _converter = &converter{}
// 	db, _, err := sqlmock.New(sqlmock.ValueConverterOption(_converter))
// 	if err != nil {
// 		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
// 	}
// 	qc := NewQueryClass(sqlx.NewDb(db, "clickhouse"))
// 	assert.EqualError(t, qc.Save(&agentMsg), "Nothing to save - no query classes")
// 	_ = db.Close()
// }
