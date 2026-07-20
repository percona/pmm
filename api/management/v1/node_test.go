// Copyright (C) 2023 Percona LLC
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

package managementv1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestRegisterNodeRequestForceNewAgentTokenJSON(t *testing.T) {
	data, err := protojson.Marshal(&RegisterNodeRequest{ForceNewAgentToken: true})
	require.NoError(t, err)
	assert.JSONEq(t, `{"forceNewAgentToken":true}`, string(data))

	var request RegisterNodeRequest
	require.NoError(t, protojson.Unmarshal(data, &request))
	assert.True(t, request.GetForceNewAgentToken())
}
