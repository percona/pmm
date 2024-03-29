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

package action

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	"github.com/percona/pmm/api/actions/v1/json/client"
	actions "github.com/percona/pmm/api/actions/v1/json/client/actions_service"
)

func TestPTSummary(t *testing.T) {
	ctx, cancel := context.WithTimeout(pmmapitests.Context, 30*time.Second)
	defer cancel()

	explainActionOK, err := client.Default.ActionsService.StartPTSummaryAction(&actions.StartPTSummaryActionParams{
		Context: ctx,
		Body: actions.StartPTSummaryActionBody{
			NodeID: "pmm-server",
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, explainActionOK.Payload.ActionID)

	for {
		actionOK, err := client.Default.ActionsService.GetAction(&actions.GetActionParams{
			Context:  ctx,
			ActionID: explainActionOK.Payload.ActionID,
		})
		require.NoError(t, err)

		if !actionOK.Payload.Done {
			time.Sleep(1 * time.Second)

			continue
		}

		require.True(t, actionOK.Payload.Done)
		require.Empty(t, actionOK.Payload.Error)
		require.NotEmpty(t, actionOK.Payload.Output)
		t.Log(actionOK.Payload.Output)

		break
	}
}
