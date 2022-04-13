package depstests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestDuration(t *testing.T) {
	// https://google.golang.org/protobuf/issues/883
	// https://google.golang.org/protobuf/issues/1219
	// https://jira.percona.com/browse/PMM-6760

	s, err := protojson.Marshal(durationpb.New(-time.Nanosecond))
	require.NoError(t, err)
	assert.Equal(t, `"-0.000000001s"`, string(s))
}
