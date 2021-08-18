// pmm-managed
// Copyright (C) 2017 Percona LLC
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

	"github.com/AlekSi/pointer"
	managementClient "github.com/percona/pmm/api/managementpb/json/client"
	"github.com/percona/pmm/api/managementpb/json/client/security_checks"
	serverClient "github.com/percona/pmm/api/serverpb/json/client"
	"github.com/percona/pmm/api/serverpb/json/client/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm-managed/api-tests"
)

func TestStartChecks(t *testing.T) {
	client := serverClient.Default.Server

	t.Run("with enabled STT", func(t *testing.T) {
		defer restoreSettingsDefaults(t)
		// Enabled STT
		res, err := client.ChangeSettings(&server.ChangeSettingsParams{
			Body: server.ChangeSettingsBody{
				EnableStt:       true,
				EnableTelemetry: true,
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.True(t, res.Payload.Settings.SttEnabled)
		assert.True(t, res.Payload.Settings.TelemetryEnabled)

		resp, err := managementClient.Default.SecurityChecks.StartSecurityChecks(nil)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("with disabled STT", func(t *testing.T) {
		defer restoreSettingsDefaults(t)
		// Disabled STT
		res, err := client.ChangeSettings(&server.ChangeSettingsParams{
			Body: server.ChangeSettingsBody{
				DisableStt:      true,
				EnableTelemetry: true,
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.False(t, res.Payload.Settings.SttEnabled)
		assert.True(t, res.Payload.Settings.TelemetryEnabled)

		resp, err := managementClient.Default.SecurityChecks.StartSecurityChecks(nil)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, `STT is disabled.`)
		assert.Nil(t, resp)
	})
}

func TestGetSecurityCheckResults(t *testing.T) {
	if !pmmapitests.RunSTTTests {
		t.Skip("Skipping STT tests until we have environment: https://jira.percona.com/browse/PMM-5106")
	}

	client := serverClient.Default.Server

	t.Run("with disabled STT", func(t *testing.T) {
		defer restoreSettingsDefaults(t)
		// Disabled STT
		res, err := client.ChangeSettings(&server.ChangeSettingsParams{
			Body: server.ChangeSettingsBody{
				DisableStt: true,
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.False(t, res.Payload.Settings.SttEnabled)

		results, err := managementClient.Default.SecurityChecks.GetSecurityCheckResults(nil)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, `STT is disabled.`)
		assert.Nil(t, results)
	})

	t.Run("with enabled STT", func(t *testing.T) {
		defer restoreSettingsDefaults(t)
		// Enabled STT
		res, err := client.ChangeSettings(&server.ChangeSettingsParams{
			Body: server.ChangeSettingsBody{
				EnableStt: true,
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.True(t, res.Payload.Settings.SttEnabled)

		resp, err := managementClient.Default.SecurityChecks.StartSecurityChecks(nil)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		results, err := managementClient.Default.SecurityChecks.GetSecurityCheckResults(nil)
		require.NoError(t, err)
		assert.NotNil(t, results)
	})
}

func TestListSecurityChecks(t *testing.T) {
	client := serverClient.Default.Server

	defer restoreSettingsDefaults(t)
	// Enable STT
	res, err := client.ChangeSettings(&server.ChangeSettingsParams{
		Body: server.ChangeSettingsBody{
			EnableStt: true,
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	assert.True(t, res.Payload.Settings.SttEnabled)

	resp, err := managementClient.Default.SecurityChecks.ListSecurityChecks(nil)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Payload.Checks)
	for _, c := range resp.Payload.Checks {
		assert.NotEmpty(t, c.Name, "%+v", c)
		assert.NotEmpty(t, c.Summary, "%+v", c)
		assert.NotEmpty(t, c.Description, "%+v", c)
	}
}

func TestChangeSecurityChecks(t *testing.T) {
	client := serverClient.Default.Server

	t.Run("enable disable", func(t *testing.T) {
		defer restoreSettingsDefaults(t)
		// Enable STT
		res, err := client.ChangeSettings(&server.ChangeSettingsParams{
			Body: server.ChangeSettingsBody{
				EnableStt: true,
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.True(t, res.Payload.Settings.SttEnabled)

		resp, err := managementClient.Default.SecurityChecks.ListSecurityChecks(nil)
		require.NoError(t, err)
		require.NotEmpty(t, resp.Payload.Checks)

		var check *security_checks.ChecksItems0

		// enable ⥁ disable loop, it checks current state of first returned check and changes its state,
		// then in second iteration it returns state to its origin.
		for i := 0; i < 2; i++ {
			check = resp.Payload.Checks[0]
			params := &security_checks.ChangeSecurityChecksParams{
				Body: security_checks.ChangeSecurityChecksBody{
					Params: []*security_checks.ParamsItems0{
						{
							Name:    check.Name,
							Disable: !check.Disabled,
							Enable:  check.Disabled,
						},
					},
				},
				Context: pmmapitests.Context,
			}

			_, err = managementClient.Default.SecurityChecks.ChangeSecurityChecks(params)
			require.NoError(t, err)

			resp, err = managementClient.Default.SecurityChecks.ListSecurityChecks(nil)
			require.NoError(t, err)
			require.NotEmpty(t, resp.Payload.Checks)

			for _, c := range resp.Payload.Checks {
				if c.Name == check.Name {
					assert.Equal(t, !check.Disabled, c.Disabled)
					break
				}
			}
		}
	})

	t.Run("change interval error", func(t *testing.T) {
		defer restoreSettingsDefaults(t)
		// Enable STT
		res, err := client.ChangeSettings(&server.ChangeSettingsParams{
			Body: server.ChangeSettingsBody{
				EnableStt: true,
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.True(t, res.Payload.Settings.SttEnabled)

		resp, err := managementClient.Default.SecurityChecks.ListSecurityChecks(nil)
		require.NoError(t, err)
		require.NotEmpty(t, resp.Payload.Checks)
		assert.Equal(t, "STANDARD", *resp.Payload.Checks[0].Interval)

		check := resp.Payload.Checks[0]
		interval := "unknown_interval"
		params := &security_checks.ChangeSecurityChecksParams{
			Body: security_checks.ChangeSecurityChecksBody{
				Params: []*security_checks.ParamsItems0{
					{
						Name:     check.Name,
						Interval: &interval,
					},
				},
			},
			Context: pmmapitests.Context,
		}

		_, err = managementClient.Default.SecurityChecks.ChangeSecurityChecks(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "unknown value \"\\\"unknown_interval\\\"\" for enum management.SecurityCheckInterval")
	})

	t.Run("change interval normal", func(t *testing.T) {
		defer restoreSettingsDefaults(t)
		defer restoreCheckIntervalDefaults(t)
		// Enable STT
		res, err := client.ChangeSettings(&server.ChangeSettingsParams{
			Body: server.ChangeSettingsBody{
				EnableStt: true,
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.True(t, res.Payload.Settings.SttEnabled)

		resp, err := managementClient.Default.SecurityChecks.ListSecurityChecks(nil)
		require.NoError(t, err)
		require.NotEmpty(t, resp.Payload.Checks)
		assert.Equal(t, "STANDARD", string(*resp.Payload.Checks[0].Interval))

		// convert all checks to RARE interval
		for _, check := range resp.Payload.Checks {
			params := &security_checks.ChangeSecurityChecksParams{
				Body: security_checks.ChangeSecurityChecksBody{
					Params: []*security_checks.ParamsItems0{
						{
							Name:     check.Name,
							Interval: pointer.ToString(security_checks.ParamsItems0IntervalRARE),
						},
					},
				},
				Context: pmmapitests.Context,
			}

			_, err = managementClient.Default.SecurityChecks.ChangeSecurityChecks(params)
			require.NoError(t, err)
		}

		resp, err = managementClient.Default.SecurityChecks.ListSecurityChecks(nil)
		require.NoError(t, err)
		require.NotEmpty(t, resp.Payload.Checks)

		for _, check := range resp.Payload.Checks {
			assert.Equal(t, "RARE", *check.Interval)
		}

		t.Run("intervals should be preserved on restart", func(t *testing.T) {
			// Enable STT
			res, err := client.ChangeSettings(&server.ChangeSettingsParams{
				Body: server.ChangeSettingsBody{
					EnableStt: true,
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			assert.True(t, res.Payload.Settings.SttEnabled)

			_, err = managementClient.Default.SecurityChecks.StartSecurityChecks(nil)
			require.NoError(t, err)

			resp, err := managementClient.Default.SecurityChecks.ListSecurityChecks(nil)
			require.NoError(t, err)
			require.NotEmpty(t, resp.Payload.Checks)
			assert.Equal(t, "RARE", *resp.Payload.Checks[0].Interval)
		})
	})
}
