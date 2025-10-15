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

// Package server contains PMM server API tests.
package server

import (
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	advisorsv1 "github.com/percona/pmm/api/advisors/v1"
	advisorClient "github.com/percona/pmm/api/advisors/v1/json/client"
	advisor "github.com/percona/pmm/api/advisors/v1/json/client/advisor_service"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	server "github.com/percona/pmm/api/server/v1/json/client/server_service"
)

func restoreSettingsDefaults(t *testing.T) {
	t.Helper()

	res, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
		Body: server.ChangeSettingsBody{
			EnableAdvisor:   pointer.ToBool(true),
			EnableTelemetry: pointer.ToBool(true),
			EnableAlerting:  pointer.ToBool(true),
			EnableUpdates:   pointer.ToBool(true),
			MetricsResolutions: &server.ChangeSettingsParamsBodyMetricsResolutions{
				Hr: "5s",
				Mr: "10s",
				Lr: "60s",
			},
			AdvisorRunIntervals: &server.ChangeSettingsParamsBodyAdvisorRunIntervals{
				FrequentInterval: "14400s",
				StandardInterval: "86400s",
				RareInterval:     "280800s",
			},
			DataRetention: "2592000s",
			AWSPartitions: &server.ChangeSettingsParamsBodyAWSPartitions{
				Values: []string{"aws"},
			},
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	assert.True(t, res.Payload.Settings.TelemetryEnabled)
	assert.True(t, res.Payload.Settings.AdvisorEnabled)
	expectedResolutions := &server.ChangeSettingsOKBodySettingsMetricsResolutions{
		Hr: "5s",
		Mr: "10s",
		Lr: "60s",
	}
	assert.Equal(t, expectedResolutions, res.Payload.Settings.MetricsResolutions)
	expectedAdvisorRunIntervals := &server.ChangeSettingsOKBodySettingsAdvisorRunIntervals{
		FrequentInterval: "14400s",
		StandardInterval: "86400s",
		RareInterval:     "280800s",
	}
	assert.Equal(t, expectedAdvisorRunIntervals, res.Payload.Settings.AdvisorRunIntervals)
	assert.Equal(t, "2592000s", res.Payload.Settings.DataRetention)
	assert.Equal(t, []string{"aws"}, res.Payload.Settings.AWSPartitions)
}

func restoreCheckIntervalDefaults(t *testing.T) {
	t.Helper()

	resp, err := advisorClient.Default.AdvisorService.ListAdvisorChecks(nil)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Payload.Checks)

	var params *advisor.ChangeAdvisorChecksParams

	for _, check := range resp.Payload.Checks {
		params = &advisor.ChangeAdvisorChecksParams{
			Body: advisor.ChangeAdvisorChecksBody{
				Params: []*advisor.ChangeAdvisorChecksParamsBodyParamsItems0{
					{
						Name:     check.Name,
						Interval: pointer.ToString(advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_STANDARD.String()),
					},
				},
			},
			Context: pmmapitests.Context,
		}

		_, err = advisorClient.Default.AdvisorService.ChangeAdvisorChecks(params)
		require.NoError(t, err)
	}
}
