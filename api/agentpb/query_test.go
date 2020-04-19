package agentpb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryResultsSerialization(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		expected := []map[string]interface{}{
			{},

			// non-zero values
			{
				"bool":  true,
				"int64": int64(-1), "uint64": uint64(1),
				"double": float64(7.42),
				"string": "funyarinpa",
				"bytes":  []byte{0x00, 0x01, 0xfe, 0xff},
				"time":   time.Now().UTC(),
				"slice":  []interface{}{int64(1), int64(2), int64(3)},
				"map":    map[string]interface{}{"k": int64(42)},
			},

			// zero values
			{
				"nil":   nil,
				"bool":  false,
				"int64": int64(0), "uint64": uint64(0),
				"double": float64(0),
				"string": "",
				"bytes1": []byte{}, "bytes2": []byte{0},
				"time":   time.Time{},
				"slice1": []interface{}{}, "slice2": []interface{}{int64(0), int64(0), int64(0)},
				"map1": map[string]interface{}{}, "map2": map[string]interface{}{"": int64(0)},
			},
		}

		b, err := MarshalActionQueryResult(expected)
		require.NoError(t, err)

		actual, err := UnmarshalActionQueryResult(b)
		require.NoError(t, err)

		assert.Equal(t, actual, expected)
	})

	t.Run("Conversions", func(t *testing.T) {
		b, err := MarshalActionQueryResult([]map[string]interface{}{
			// non-zero values
			{
				"int": int(-1), "uint": uint(1),
				"double": float32(7.42),
				"slice":  []int{1, 2, 3},
				"map":    map[string]int{"k": 42},
			},

			// zero values
			{
				// "nil": (*int)(nil), - do we need it?
				"int": int(0), "uint": uint(0),
				"double": float32(0),
				"slice1": []int{}, "slice2": []int{0},
				"map1": map[string]int{}, "map2": map[string]int{"": 0},
			},
		})
		require.NoError(t, err)

		actual, err := UnmarshalActionQueryResult(b)
		require.NoError(t, err)

		expected := []map[string]interface{}{
			{
				"int": int64(-1), "uint": uint64(1),
				"double": float64(7.420000076293945),
				"slice":  []interface{}{int64(1), int64(2), int64(3)},
				"map":    map[string]interface{}{"k": int64(42)},
			},

			{
				"int": int64(0), "uint": uint64(0),
				"double": float64(0),
				"slice1": []interface{}{}, "slice2": []interface{}{int64(0)},
				"map1": map[string]interface{}{}, "map2": map[string]interface{}{"": int64(0)},
			},
		}
		assert.Equal(t, actual, expected)
	})
}
