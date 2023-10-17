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

package ia

import (
	"testing"

	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	channelsClient "github.com/percona/pmm/api/managementpb/ia/json/client"
	"github.com/percona/pmm/api/managementpb/ia/json/client/alerts"
)

func TestAlertsAPI(t *testing.T) {
	client := channelsClient.Default.Alerts

	t.Run("list", func(t *testing.T) {
		_, err := client.ListAlerts(&alerts.ListAlertsParams{
			Body:    alerts.ListAlertsBody{},
			Context: pmmapitests.Context,
		})

		require.NoError(t, err)
	})
}
