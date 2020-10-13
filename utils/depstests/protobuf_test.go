package depstests

import (
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuration(t *testing.T) {
	// https://github.com/golang/protobuf/issues/883
	// https://github.com/golang/protobuf/issues/1219
	// https://jira.percona.com/browse/PMM-6760

	var m jsonpb.Marshaler
	s, err := m.MarshalToString(ptypes.DurationProto(-time.Nanosecond))
	require.NoError(t, err)
	assert.Equal(t, `"-0.000000001s"`, s)
}
