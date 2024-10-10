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

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	server "github.com/percona/pmm/api/server/v1/json/client/server_service"
)

func TestPanics(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"panic-error", "panic-fmterror", "panic-string"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()

			res, err := serverClient.Default.ServerService.Version(&server.VersionParams{
				Dummy:   &mode,
				Context: pmmapitests.Context,
			})
			assert.Empty(t, res)
			pmmapitests.AssertAPIErrorf(t, err, 500, codes.Internal, "Internal server error.")
		})
	}
}
