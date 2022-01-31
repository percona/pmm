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
	"time"

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
	t.Run("with enabled STT", func(t *testing.T) {
		toggleSTT(t, true)
		t.Cleanup(func() { restoreSettingsDefaults(t) })

		resp, err := managementClient.Default.SecurityChecks.StartSecurityChecks(nil)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("with disabled STT", func(t *testing.T) {
		toggleSTT(t, false)
		t.Cleanup(func() { restoreSettingsDefaults(t) })

		resp, err := managementClient.Default.SecurityChecks.StartSecurityChecks(nil)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, `STT is disabled.`)
		assert.Nil(t, resp)
	})
}

func TestGetSecurityCheckResults(t *testing.T) {
	if !pmmapitests.RunSTTTests {
		t.Skip("Skipping STT tests until we have environment: https://jira.percona.com/browse/PMM-5106")
	}

	t.Run("with disabled STT", func(t *testing.T) {
		toggleSTT(t, true)
		t.Cleanup(func() { restoreSettingsDefaults(t) })

		results, err := managementClient.Default.SecurityChecks.GetSecurityCheckResults(nil)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, `STT is disabled.`)
		assert.Nil(t, results)
	})

	t.Run("with enabled STT", func(t *testing.T) {
		toggleSTT(t, true)
		t.Cleanup(func() { restoreSettingsDefaults(t) })

		resp, err := managementClient.Default.SecurityChecks.StartSecurityChecks(nil)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		results, err := managementClient.Default.SecurityChecks.GetSecurityCheckResults(nil)
		require.NoError(t, err)
		assert.NotNil(t, results)
	})
}

func TestListSecurityChecks(t *testing.T) {
	toggleSTT(t, true)
	t.Cleanup(func() { restoreSettingsDefaults(t) })

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
	toggleSTT(t, true)
	t.Cleanup(func() { restoreSettingsDefaults(t) })

	t.Run("enable disable", func(t *testing.T) {
		t.Run("enable disable", func(t *testing.T) {
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
			t.Cleanup(func() { restoreCheckIntervalDefaults(t) })

			resp, err := managementClient.Default.SecurityChecks.ListSecurityChecks(nil)
			require.NoError(t, err)
			require.NotEmpty(t, resp.Payload.Checks)

			check := resp.Payload.Checks[0]
			interval := *check.Interval
			params := &security_checks.ChangeSecurityChecksParams{
				Body: security_checks.ChangeSecurityChecksBody{
					Params: []*security_checks.ParamsItems0{
						{
							Name:     check.Name,
							Interval: pointer.ToString("unknown_interval"),
						},
					},
				},
				Context: pmmapitests.Context,
			}

			_, err = managementClient.Default.SecurityChecks.ChangeSecurityChecks(params)
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "unknown value \"\\\"unknown_interval\\\"\" for enum management.SecurityCheckInterval")

			resp, err = managementClient.Default.SecurityChecks.ListSecurityChecks(nil)
			require.NoError(t, err)
			require.NotEmpty(t, resp.Payload.Checks)

			var found bool
			for _, c := range resp.Payload.Checks {
				if c.Name == check.Name {
					found = true
					assert.Equal(t, interval, *c.Interval)
				}
			}

			assert.True(t, found, "required check wasn't found")
		})

		t.Run("change interval normal", func(t *testing.T) {
			t.Cleanup(func() { restoreSettingsDefaults(t) })

			resp, err := managementClient.Default.SecurityChecks.ListSecurityChecks(nil)
			require.NoError(t, err)
			require.NotEmpty(t, resp.Payload.Checks)

			// convert all checks to RARE interval
			pp := make([]*security_checks.ParamsItems0, len(resp.Payload.Checks))
			for i, check := range resp.Payload.Checks {
				pp[i] = &security_checks.ParamsItems0{
					Name:     check.Name,
					Interval: pointer.ToString(security_checks.ParamsItems0IntervalRARE),
				}
			}

			params := &security_checks.ChangeSecurityChecksParams{
				Body:    security_checks.ChangeSecurityChecksBody{Params: pp},
				Context: pmmapitests.Context,
			}
			_, err = managementClient.Default.SecurityChecks.ChangeSecurityChecks(params)
			require.NoError(t, err)

			resp, err = managementClient.Default.SecurityChecks.ListSecurityChecks(nil)
			require.NoError(t, err)
			require.NotEmpty(t, resp.Payload.Checks)

			for _, check := range resp.Payload.Checks {
				assert.Equal(t, "RARE", *check.Interval)
			}

			t.Run("intervals should be preserved on restart", func(t *testing.T) {
				resp, err := managementClient.Default.SecurityChecks.ListSecurityChecks(nil)
				require.NoError(t, err)
				require.NotEmpty(t, resp.Payload.Checks)
				assert.Equal(t, "RARE", *resp.Payload.Checks[0].Interval)
			})
		})
	})
}

func toggleSTT(t *testing.T, enable bool) {
	t.Helper()

	res, err := serverClient.Default.Server.ChangeSettings(&server.ChangeSettingsParams{
		Body: server.ChangeSettingsBody{
			EnableStt:  enable,
			DisableStt: !enable,
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	require.Equal(t, enable, res.Payload.Settings.SttEnabled)

	if enable {
		// It takes some time to load check files
		time.Sleep(time.Second)
	}
}
