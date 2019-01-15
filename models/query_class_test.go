package models

import (
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"net"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	_ "github.com/kshvakov/clickhouse"
	"github.com/kshvakov/clickhouse/lib/column"
	"github.com/kshvakov/clickhouse/lib/types"
	"github.com/stretchr/testify/assert"

	collectorpb "github.com/Percona-Lab/qan-api/api/collector"
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

func TestSave(t *testing.T) {
	agentMsg := collectorpb.AgentMessage{
		QueryClass: []*collectorpb.QueryClass{
			{
				Digest:   "digest1",
				Labels:   map[string]string{"label1": "aaa1"},
				Warnings: map[string]uint64{"warn1": 111},
				Errors:   map[string]uint64{"error1": 333},
				Labint:   map[uint32]uint32{321: 123},
			},
			{
				Digest:   "digest2",
				Labels:   map[string]string{"label2": "bbb2"},
				Warnings: map[string]uint64{"warn2": 222},
				Errors:   map[string]uint64{"error1": 444},
				Labint:   map[uint32]uint32{987: 789},
			},
		},
	}
	var _converter = &converter{}
	db, mock, err := sqlmock.New(sqlmock.ValueConverterOption(_converter))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	mock.ExpectBegin()
	a := mock.ExpectPrepare("^INSERT INTO queries .*")
	for _, qc := range agentMsg.QueryClass {
		s := reflect.ValueOf(*qc)
		ret := make([]driver.Value, s.NumField())
		for i := 0; i < s.NumField(); i++ {
			ret[i] = Any{}
		}
		ret[0] = qc.Digest
		ret[6], ret[7] = MapToArrsStrStr(qc.Labels)     // Query class labels.
		ret[17], ret[18] = MapToArrsStrInt(qc.Warnings) // Query class warnings.
		ret[20], ret[21] = MapToArrsStrInt(qc.Errors)   // Query class errors.
		ret[148], ret[149] = MapToArrsIntInt(qc.Labint) // Query class labint.
		a.ExpectExec().WithArgs(ret...).WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit()
	qc := NewQueryClass(sqlx.NewDb(db, "clickhouse"))

	// execute save method
	if err = qc.Save(&agentMsg); err != nil {
		t.Errorf("error was not expected while saving data to clickhouse: %s", err)
	}

	_ = db.Close()
	// we make sure that all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestSaveEpmtyMaps(t *testing.T) {
	agentMsg := collectorpb.AgentMessage{
		QueryClass: []*collectorpb.QueryClass{
			{
				Digest: "digest1",
			},
			{
				Digest: "digest2",
			},
		},
	}
	var _converter = &converter{}
	db, mock, err := sqlmock.New(sqlmock.ValueConverterOption(_converter))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	mock.ExpectBegin()
	a := mock.ExpectPrepare("^INSERT INTO queries .*")
	for _, qc := range agentMsg.QueryClass {
		s := reflect.ValueOf(*qc)
		ret := make([]driver.Value, s.NumField())
		for i := 0; i < s.NumField(); i++ {
			ret[i] = Any{}
		}
		ret[0] = qc.Digest
		a.ExpectExec().WithArgs(ret...).WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit()
	qc := NewQueryClass(sqlx.NewDb(db, "clickhouse"))

	// execute save method
	if err = qc.Save(&agentMsg); err != nil {
		t.Errorf("error was not expected while saving data to clickhouse: %s", err)
	}

	_ = db.Close()
	// we make sure that all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestSaveEpmtyQueryClass(t *testing.T) {
	agentMsg := collectorpb.AgentMessage{
		QueryClass: []*collectorpb.QueryClass{},
	}
	var _converter = &converter{}
	db, _, err := sqlmock.New(sqlmock.ValueConverterOption(_converter))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	qc := NewQueryClass(sqlx.NewDb(db, "clickhouse"))
	assert.EqualError(t, qc.Save(&agentMsg), "Nothing to save - no query classes")
	_ = db.Close()
}
