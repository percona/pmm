package agentpb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestQuerySQLResultsSerialization(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		now := time.Now().UTC().Round(0) // strip monotonic clock reading
		columns := []string{
			"bool",
			"int64",
			"uint64",
			"double",
			"string",
			"bytes",
			"time",
			"slice",
			"map",
		}
		rows := [][]interface{}{
			// non-zero values
			{
				true,
				int64(-1),
				uint64(1),
				float64(7.42),
				"\x00\x01\xfe\xff",
				[]byte{0x00, 0x01, 0xfe, 0xff},
				now,
				[]interface{}{int64(1), int64(2), int64(3)},
				map[string]interface{}{"k": int64(42)},
			},

			// zero values
			{
				false,
				int64(0),
				uint64(0),
				float64(0),
				"",
				[]byte{},
				time.Time{},
				[]interface{}{},
				map[string]interface{}{},
			},

			// other cases
			{
				false,
				int64(0),
				uint64(0),
				float64(0),
				"\x00",
				[]byte{0x00},
				time.Time{},
				[]interface{}{int64(0), int64(0), int64(0)},
				map[string]interface{}{"": int64(0)},
			},
		}
		expected := []map[string]interface{}{
			// non-zero values
			{
				"bool":   true,
				"int64":  int64(-1),
				"uint64": uint64(1),
				"double": float64(7.42),
				"string": "\x00\x01\xfe\xff",
				"bytes":  "\x00\x01\xfe\xff",
				"time":   now,
				"slice":  []interface{}{int64(1), int64(2), int64(3)},
				"map":    map[string]interface{}{"k": int64(42)},
			},

			// zero values
			{
				"bool":   false,
				"int64":  int64(0),
				"uint64": uint64(0),
				"double": float64(0),
				"string": "",
				"bytes":  "",
				"time":   time.Time{},
				"slice":  []interface{}{},
				"map":    map[string]interface{}{},
			},

			// other cases
			{
				"bool":   false,
				"int64":  int64(0),
				"uint64": uint64(0),
				"double": float64(0),
				"string": "\x00",
				"bytes":  "\x00",
				"time":   time.Time{},
				"slice":  []interface{}{int64(0), int64(0), int64(0)},
				"map":    map[string]interface{}{"": int64(0)},
			},
		}

		b, err := MarshalActionQuerySQLResult(columns, rows)
		require.NoError(t, err)

		actual, err := UnmarshalActionQueryResult(b)
		require.NoError(t, err)

		assert.Equal(t, expected, actual)
	})

	t.Run("InvalidColumns", func(t *testing.T) {
		columns := []string{"foo"}
		rows := [][]interface{}{{}}

		_, err := MarshalActionQuerySQLResult(columns, rows)
		require.EqualError(t, err, "invalid result: expected 1 columns in row 0, got 0")
	})
}

func TestQueryDocsResultsSerialization(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		now := time.Now().UTC().Round(0) // strip monotonic clock reading
		expected := []map[string]interface{}{
			{},

			// non-zero values
			{
				"bool":   true,
				"int64":  int64(-1),
				"uint64": uint64(1),
				"double": float64(7.42),
				"string": "\x00\x01\xfe\xff",
				"time":   now,
				"slice":  []interface{}{int64(1), int64(2), int64(3)},
				"map":    map[string]interface{}{"k": int64(42)},
			},

			// zero values
			{
				"nil":     nil,
				"bool":    false,
				"int64":   int64(0),
				"uint64":  uint64(0),
				"double":  float64(0),
				"string1": "", "string2": "\x00",
				"time":   time.Time{},
				"slice1": []interface{}{}, "slice2": []interface{}{int64(0), int64(0), int64(0)},
				"map1": map[string]interface{}{}, "map2": map[string]interface{}{"": int64(0)},
			},
		}

		b, err := MarshalActionQueryDocsResult(expected)
		require.NoError(t, err)

		actual, err := UnmarshalActionQueryResult(b)
		require.NoError(t, err)

		assert.Equal(t, expected, actual)
	})

	t.Run("Conversions", func(t *testing.T) {
		now := time.Now().UTC().Round(0) // strip monotonic clock reading
		b, err := MarshalActionQueryDocsResult([]map[string]interface{}{
			// non-zero values
			{
				"int": int(-1), "int8": int8(-1), "int16": int16(-1), "int32": int32(-1),
				"uint": uint(1), "uint8": uint8(1), "uint16": uint16(1), "uint32": uint32(1),
				"double": float32(7.42),
				"bytes1": []byte("funyarinpa"), "bytes2": []byte{0x00, 0x01, 0xfe, 0xff},
				"mongoTimestamp": primitive.Timestamp{T: uint32(now.Unix()), I: 42},
				"mongoDateTime":  primitive.NewDateTimeFromTime(now),
				"slice":          []int{1, 2, 3},
				"map":            map[string]int{"k": 42},
			},

			// zero values
			{
				// "nil": (*int)(nil), - do we need it?
				"int": int(0), "int8": int8(0), "int16": int16(0), "int32": int32(0),
				"uint": uint(0), "uint8": uint8(0), "uint16": uint16(0), "uint32": uint32(0),
				"double":         float32(0),
				"bytes":          []byte{},
				"mongoTimestamp": primitive.Timestamp{},
				"mongoDateTime":  primitive.DateTime(0),
				"slice1":         []int{}, "slice2": []int{0},
				"map1": map[string]int{}, "map2": map[string]int{"": 0},
			},
		})
		require.NoError(t, err)

		actual, err := UnmarshalActionQueryResult(b)
		require.NoError(t, err)

		expected := []map[string]interface{}{
			{
				"int": int64(-1), "int8": int64(-1), "int16": int64(-1), "int32": int64(-1),
				"uint": uint64(1), "uint8": uint64(1), "uint16": uint64(1), "uint32": uint64(1),
				"double": float64(7.420000076293945),
				"bytes1": "funyarinpa", "bytes2": "\x00\x01\xfe\xff",
				"mongoTimestamp": now.Truncate(time.Second).Add(42 * time.Nanosecond), // resolution is up to a second; cram I (ordinal) into nanoseconds
				"mongoDateTime":  now.Truncate(time.Millisecond),                      // resolution is up to a millisecond
				"slice":          []interface{}{int64(1), int64(2), int64(3)},
				"map":            map[string]interface{}{"k": int64(42)},
			},

			{
				"int": int64(0), "int8": int64(0), "int16": int64(0), "int32": int64(0),
				"uint": uint64(0), "uint8": uint64(0), "uint16": uint64(0), "uint32": uint64(0),
				"double":         float64(0),
				"bytes":          "",
				"mongoTimestamp": time.Time{},
				"mongoDateTime":  time.Time{},
				"slice1":         []interface{}{}, "slice2": []interface{}{int64(0)},
				"map1": map[string]interface{}{}, "map2": map[string]interface{}{"": int64(0)},
			},
		}
		assert.Equal(t, expected, actual)
	})
}
