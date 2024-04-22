// Copyright (C) 2024 Percona LLC
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
