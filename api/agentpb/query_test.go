package agentpb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryResultsSerialization(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		expected := []map[string]interface{}{
			{},

			// non-zero values
			{
				"bool": true, "int64": int64(-1), "uint64": uint64(1),
				"map": map[string]int64{"k": int64(42)},
			},

			// zero values
			{
				"nil":  nil,
				"bool": false, "int64": int64(0), "uint64": uint64(0),
				"map1": map[string]int64{}, "map2": map[string]int64{"": int64(0)},
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
			},

			// zero values
			{
				"int": int(0), "uint": uint(0),
			},
		})
		require.NoError(t, err)

		actual, err := UnmarshalActionQueryResult(b)
		require.NoError(t, err)

		expected := []map[string]interface{}{
			{"int": int64(-1), "uint": uint64(1)},
			{"int": int64(0), "uint": uint64(0)},
		}
		assert.Equal(t, actual, expected)
	})
}
