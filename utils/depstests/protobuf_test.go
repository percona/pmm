// pmm-agent
// Copyright 2019 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package depstests

import (
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb" //nolint:staticcheck
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestDuration(t *testing.T) {
	// https://github.com/golang/protobuf/issues/883
	// https://github.com/golang/protobuf/issues/1219
	// https://jira.percona.com/browse/PMM-6760

	var m jsonpb.Marshaler
	s, err := m.MarshalToString(durationpb.New(-time.Nanosecond))
	require.NoError(t, err)
	assert.Equal(t, `"-0.000000001s"`, s)
}
