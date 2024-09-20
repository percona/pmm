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

package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	"github.com/percona/pmm/api/management/v1/json/client"
	mgmtSvc "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

func TestListAgentVersions(t *testing.T) {
	ctx, cancel := context.WithTimeout(pmmapitests.Context, 30*time.Second)
	t.Cleanup(func() { cancel() })

	t.Run("PMM Agent needs no update", func(t *testing.T) {
		resp, err := client.Default.ManagementService.ListAgentVersions(
			&mgmtSvc.ListAgentVersionsParams{
				Context: ctx,
			})
		require.NoError(t, err)
		require.Len(t, resp.Payload.AgentVersions, 1)

		expected := mgmtSvc.ListAgentVersionsOKBodyAgentVersionsItems0SeverityUPDATESEVERITYUPTODATE
		require.Equal(t, expected, *resp.Payload.AgentVersions[0].Severity)
	})
}
