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

package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientDebugOff(t *testing.T) {
	// SWAGGER_DEBUG and DEBUG are what httptransport.New reads for its debug setting.
	t.Setenv("SWAGGER_DEBUG", "1")
	t.Setenv("DEBUG", "1")

	c, err := newClient("http://127.0.0.1:1/")
	require.NoError(t, err)
	assert.False(t, c.swagger.Debug, "go-openapi's debug dump would write the caller's Authorization and cookies to its logger")
}
